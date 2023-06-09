package django

import (
	"fmt"
	"github.com/flosch/pongo2/v6"
	"github.com/xingmoo/library/utils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Engine struct
type Engine struct {
	// delimiters
	left  string
	right string
	// views folder
	directory string
	// http.FileSystem supports embedded files
	fileSystem http.FileSystem
	// views extension
	extension string
	// layout variable name that incapsulates the template
	// determines if the engine parsed all templates
	loaded bool
	// reload on each render
	reload bool
	// debug prints the parsed templates
	debug bool
	// forward the base path to the template engine
	forwardPath bool
	// lock for funcmap and templates
	mutex sync.RWMutex
	// template funcmap
	funcmap map[string]interface{}
	// templates
	Templates map[string]*pongo2.Template
}

// New returns a Django render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		left:      "{{",
		right:     "}}",
		directory: directory,
		extension: extension,
		funcmap:   make(map[string]interface{}),
	}
	return engine
}

func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	engine := &Engine{
		left:       "{{",
		right:      "}}",
		directory:  "/",
		fileSystem: fs,
		extension:  extension,
		funcmap:    make(map[string]interface{}),
	}
	return engine
}

// NewPathForwardingFileSystem Passes "directory" to the template engine where alternative functions don't.
//
//	This fixes errors during resolution of templates when "{% extends 'parent.html' %}" is used.
func NewPathForwardingFileSystem(fs http.FileSystem, directory string, extension string) *Engine {
	engine := NewFileSystem(fs, extension)
	engine.forwardPath = true
	engine.directory = directory
	return engine
}

// Delims sets the action delimiters to the specified strings, to be used in
// templates. An empty delimiter stands for the
// corresponding default: {{ or }}.
func (e *Engine) Delims(left, right string) *Engine {

	return e
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFunc(name string, fn any) *Engine {
	e.mutex.Lock()
	e.funcmap[name] = fn
	e.mutex.Unlock()
	return e
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you don't want to restart
// the application when you edit a template file.
func (e *Engine) Reload(enabled bool) *Engine {
	e.reload = enabled
	return e
}

// Debug will print the parsed templates when Load is triggered.
func (e *Engine) Debug(enabled bool) *Engine {
	e.debug = enabled
	return e
}

// Parse is deprecated, please use Load() instead
func (e *Engine) Parse() error {
	return e.Load()
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.Templates = make(map[string]*pongo2.Template)

	baseDir := e.directory

	var pongoloader pongo2.TemplateLoader
	if e.fileSystem != nil {
		// ensures creation of httpFileSystemLoader only when filesystem is defined
		if e.forwardPath {
			pongoloader = pongo2.MustNewHttpFileSystemLoader(e.fileSystem, baseDir)
		} else {
			pongoloader = pongo2.MustNewHttpFileSystemLoader(e.fileSystem, "")
		}
	} else {
		pongoloader = pongo2.MustNewLocalFileSystemLoader(baseDir)
	}

	// New pongo2 defaultset
	pongoset := pongo2.NewSet("default", pongoloader)
	// Set template settings
	pongoset.Globals.Update(e.funcmap)
	pongo2.SetAutoescape(false)

	// Loop trough each directory and register template files
	walkFn := func(path string, info os.FileInfo, err error) error {
		// Return error if exist
		if err != nil {
			return err
		}
		// Skip file if it's a directory or has no file info
		if info == nil || info.IsDir() {
			return nil
		}
		// Skip file if it does not equal the given template extension
		if len(e.extension) >= len(path) || path[len(path)-len(e.extension):] != e.extension {
			return nil
		}
		// Get the relative file path
		// ./views/html/index.tmpl -> index.tmpl
		rel, err := filepath.Rel(e.directory, path)
		if err != nil {
			return err
		}
		// Reverse slashes '\' -> '/' and
		// partials\footer.tmpl -> partials/footer.tmpl
		name := filepath.ToSlash(rel)
		// Remove ext from name 'index.tmpl' -> 'index'
		name = strings.TrimSuffix(name, e.extension)
		// name = strings.Replace(name, e.extension, "", -1)
		// Read the file
		// #gosec G304
		buf, err := utils.ReadFile(path, e.fileSystem)
		if err != nil {
			return err
		}
		// Create new template associated with the current one
		tmpl, err := pongoset.FromBytes(buf)
		if err != nil {
			return fmt.Errorf("views: parsed template：%s error，%w", rel, err)
		}
		tmpl.Options.TrimBlocks = true
		tmpl.Options.LStripBlocks = true

		e.Templates[name] = tmpl
		// Debugging
		if e.debug {
			fmt.Printf("views: parsed template: %s\n", name)
		}
		return err
	}
	// notify engine that we parsed all templates
	e.loaded = true
	if e.fileSystem != nil {
		return utils.Walk(e.fileSystem, e.directory, walkFn)
	}
	return filepath.Walk(e.directory, walkFn)
}

func getPongoBinding(binding any) pongo2.Context {
	if binding == nil {
		return nil
	}
	if binds, ok := binding.(pongo2.Context); ok {
		return binds
	}
	if binds, ok := binding.(map[string]any); ok {
		return binds
	}

	return nil
}

func (e *Engine) ReanderBytes(template string, binding any) ([]byte, error) {
	if !e.loaded || e.reload {
		if e.reload {
			e.loaded = false
		}
		if err := e.Load(); err != nil {
			return nil, err
		}
	}
	tmpl, ok := e.Templates[template]
	if !ok {
		return nil, fmt.Errorf("template %s does not exist", template)
	}

	bind := getPongoBinding(binding)

	return tmpl.ExecuteBytes(bind)

}

// Render will render the template by name

func (e *Engine) Render(out io.Writer, template string, binding any) error {

	parsed, err := e.ReanderBytes(template, binding)
	if err != nil {
		return err
	}

	if _, err = out.Write(parsed); err != nil {
		return err
	}
	return nil
}
