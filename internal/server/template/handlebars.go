package template

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/handlebars/v2"
	"github.com/mailgun/raymond/v2"
	"go.uber.org/zap"
)

// MultiHandlebars is a custom handlebars engine that supports multiple template directories
type MultiHandlebars struct {
	*handlebars.Engine
	fallbackDir string
	logger      *zap.Logger
}

// New creates a new MultiHandlebars engine
func New(viewsDir, fallbackDir string, extension string, logger *zap.Logger) *MultiHandlebars {
	// Handle @views special path
	if viewsDir == "@views" {
		logger.Info("using fallback directory as primary directory",
			zap.String("fallback_dir", fallbackDir))
		viewsDir = fallbackDir
	}

	// Get absolute paths
	absViewsDir, err := filepath.Abs(viewsDir)
	if err != nil {
		panic(fmt.Sprintf("failed to get absolute path for views directory: %v", err))
	}

	absFallbackDir, err := filepath.Abs(fallbackDir)
	if err != nil {
		panic(fmt.Sprintf("failed to get absolute path for fallback directory: %v", err))
	}

	logger.Info("initializing template engine",
		zap.String("views_dir", absViewsDir),
		zap.String("fallback_dir", absFallbackDir))

	baseEngine := handlebars.New(absViewsDir, extension)
	baseEngine.Templates = make(map[string]*raymond.Template)

	// Register helpers
	raymond.RegisterHelper("startsWith", func(str, prefix string) bool {
		return strings.HasPrefix(str, prefix)
	})

	raymond.RegisterHelper("or", func(args ...interface{}) bool {
		for _, arg := range args {
			// Convert to boolean and check if true
			switch v := arg.(type) {
			case bool:
				if v {
					return true
				}
			case string:
				if v != "" {
					return true
				}
			case int, int64, float64:
				if v != 0 {
					return true
				}
			default:
				if v != nil {
					return true
				}
			}
		}
		return false
	})

	raymond.RegisterHelper("eq", func(a, b interface{}) bool {
		return a == b
	})

	engine := &MultiHandlebars{
		Engine:      baseEngine,
		fallbackDir: absFallbackDir,
		logger:      logger,
	}

	// Load templates immediately
	if err := engine.Load(); err != nil {
		panic(fmt.Sprintf("failed to load templates: %v", err))
	}

	return engine
}

// loadFromDir loads templates from a directory
func (e *MultiHandlebars) loadFromDir(dir string) error {
	// Skip if directory doesn't exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		e.logger.Warn("directory does not exist, skipping", zap.String("dir", dir))
		return nil
	}

	e.logger.Info("loading templates from directory", zap.String("dir", dir))

	// Walk the directory
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a file or doesn't have the right extension
		if info.IsDir() || !strings.HasSuffix(path, e.Engine.Extension) {
			return nil
		}

		// Get relative path from the directory
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip if template already exists (for fallback directory)
		if e.Engine.Templates[rel] != nil {
			e.logger.Debug("template already exists, skipping",
				zap.String("path", path),
				zap.String("rel_path", rel))
			return nil
		}

		// Read the file
		buf, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Parse the template
		tmpl, err := raymond.Parse(string(buf))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		// Register partials
		if strings.Contains(rel, "partials/") {
			name := strings.TrimSuffix(rel, e.Engine.Extension)
			raymond.RegisterPartial(name, string(buf))
			e.logger.Debug("registered partial",
				zap.String("name", name),
				zap.String("path", path))
		}

		// Register layouts
		if strings.Contains(rel, "layouts/") {
			name := strings.TrimSuffix(rel, e.Engine.Extension)
			e.Engine.Templates[name] = tmpl
			e.logger.Debug("registered layout",
				zap.String("name", name),
				zap.String("path", path))
		}

		// Register regular templates
		e.Engine.Templates[rel] = tmpl
		e.logger.Debug("registered template",
			zap.String("rel_path", rel),
			zap.String("path", path))

		return nil
	})
}

// Load implements the template.Engine interface
func (e *MultiHandlebars) Load() error {
	e.logger.Info("loading templates")

	// First load templates from the primary directory
	if err := e.loadFromDir(e.Engine.Directory); err != nil {
		e.logger.Error("failed to load templates from primary directory",
			zap.String("dir", e.Engine.Directory),
			zap.Error(err))
		return err
	}

	// Then load templates from the fallback directory
	// Only load templates that don't exist in the primary directory
	if err := e.loadFromDir(e.fallbackDir); err != nil {
		e.logger.Error("failed to load templates from fallback directory",
			zap.String("dir", e.fallbackDir),
			zap.Error(err))
		return err
	}

	// Verify that we have all required templates
	requiredTemplates := []string{
		"index.hbs",
		"layouts/main.hbs",
		"partials/head.hbs",
	}

	missingTemplates := []string{}
	for _, tmpl := range requiredTemplates {
		if e.Engine.Templates[tmpl] == nil {
			missingTemplates = append(missingTemplates, tmpl)
		}
	}

	if len(missingTemplates) > 0 {
		e.logger.Error("missing required templates",
			zap.Strings("missing", missingTemplates))
		return fmt.Errorf("missing required templates: %v", missingTemplates)
	}

	e.logger.Info("finished loading templates",
		zap.Int("total_templates", len(e.Engine.Templates)))

	// Log all loaded templates at debug level
	templates := make([]string, 0, len(e.Engine.Templates))
	for name := range e.Engine.Templates {
		templates = append(templates, name)
	}
	e.logger.Debug("loaded templates", zap.Strings("templates", templates))

	return nil
}

// Render implements the template.Engine interface
func (e *MultiHandlebars) Render(out io.Writer, template string, binding interface{}, layout ...string) error {
	e.logger.Debug("rendering template",
		zap.String("template", template),
		zap.Any("layout", layout),
		zap.Any("binding", binding))

	// Get the template
	tmpl := e.Engine.Templates[template+e.Engine.Extension]
	if tmpl == nil {
		e.logger.Error("template not found",
			zap.String("template", template),
			zap.String("extension", e.Engine.Extension))
		return fmt.Errorf("template %s not found", template)
	}

	// If layout is specified, wrap the content in the layout
	var content interface{}
	if len(layout) > 0 && layout[0] != "" {
		layoutName := layout[0] + e.Engine.Extension
		layoutTmpl := e.Engine.Templates[layoutName]
		if layoutTmpl == nil {
			e.logger.Error("layout not found",
				zap.String("layout", layoutName))
			return fmt.Errorf("layout %s not found", layoutName)
		}

		// Execute the main template first
		result, err := tmpl.Exec(binding)
		if err != nil {
			e.logger.Error("failed to execute template",
				zap.String("template", template),
				zap.Error(err))
			return err
		}

		// Create layout binding with the template result
		layoutBinding := fiber.Map{
			"embed": raymond.SafeString(result),
		}
		// Add all original binding values to layout binding
		if m, ok := binding.(fiber.Map); ok {
			for k, v := range m {
				layoutBinding[k] = v
			}
		}

		content = layoutBinding
		tmpl = layoutTmpl
	} else {
		content = binding
	}

	// Execute the final template
	result, err := tmpl.Exec(content)
	if err != nil {
		e.logger.Error("failed to execute template",
			zap.String("template", template),
			zap.Error(err))
		return err
	}

	_, err = out.Write([]byte(result))
	return err
}
