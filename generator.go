package thriftgo_tools

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/cloudwego/thriftgo/generator/backend"
	"github.com/cloudwego/thriftgo/generator/golang"
	"github.com/cloudwego/thriftgo/parser"
	"github.com/cloudwego/thriftgo/plugin"
)

var Version = "0.0.1"

type Desc struct {
	Version string
	PkgPath string // package path like "github.com/cloudwego/thriftgo"
	PkgName string // package name like "main"
	Imports []string
}

func (d Desc) getTypeName(typeName string) string {
	if d.PkgPath == "" || strings.Contains(typeName, ".") {
		return typeName
	}
	return d.PkgName + "." + typeName
}

type ServiceDesc struct {
	Desc
	ServiceTypeName     string
	ServiceFunctionName string
	Handlers            []HandlerDesc
}

type HandlerDesc struct {
	HTTPMethod       string
	Route            string
	HandlerFuncName  string
	RequestTypeName  string
	ResponseTypeName string
}

type Args struct {
	HandlerPath   string
	RouterPath    string
	ServicePath   string
	Module        string
	PackagePrefix string
	TemplateDir   string
}

type Generator struct {
	logfunc       backend.LogFunc
	warns         []string
	resp          *plugin.Response
	codeutils     *golang.CodeUtils
	handlerTpl    *template.Template
	routerTpl     *template.Template
	serviceTpl    *template.Template
	routerBodyTpl *template.Template
	tplFuncs      template.FuncMap
}

func NewGenerator() *Generator {
	g := &Generator{
		resp:  new(plugin.Response),
		warns: make([]string, 0),
	}
	g.logfunc = backend.LogFunc{
		Info:      func(v ...interface{}) { g.warns = append(g.warns, fmt.Sprintf("%s", v...)) },
		Warn:      func(v ...interface{}) { g.warns = append(g.warns, fmt.Sprintf("%s", v...)) },
		MultiWarn: func(warns []string) { g.warns = append(g.warns, warns...) },
	}
	g.codeutils = golang.NewCodeUtils(g.logfunc)
	g.tplFuncs = template.FuncMap{
		"InsertionPoint": plugin.InsertionPoint,
	}
	return g
}

func (g *Generator) parseStructFieldAnnotation(annotations parser.Annotations) (string, error) {
	tags := make([]string, 0)
	for _, a := range annotations {
		if strings.HasPrefix(a.Key, "api.") {
			switch strings.ToUpper(a.Key[4:]) {
			case "PATH", "QUERY", "FORM", "COOKIE", "HEADER", "BODY", "VD":
				tags = append(tags, fmt.Sprintf("%s:\"%s\"", a.Key[4:], strings.Join(a.Values, ",")))
			default:
				return "", fmt.Errorf("annotations %s is not support", a.Key)
			}
		}
	}

	return strings.Join(tags, " "), nil
}

func (g *Generator) parseServiceFuncAnnotation(annotations parser.Annotations, handler *HandlerDesc) error {
	for _, a := range annotations {
		if strings.HasPrefix(a.Key, "api.") {
			method := strings.ToUpper(a.Key[4:])
			switch method {
			case "GET", "PUT", "POST", "DELETE":
				handler.HTTPMethod = method
				handler.Route = a.Values[0]
			default:
				return fmt.Errorf("annotations %s is not support", a.Key)
			}
		}
	}
	return nil
}

func (g *Generator) genPatchs(scope *golang.Scope) ([]*plugin.Generated, error) {
	patchs := make([]*plugin.Generated, 0)
	for _, sl := range scope.StructLikes() {
		for _, f := range sl.Fields() {
			tag, err := g.parseStructFieldAnnotation(f.Annotations)
			if err != nil {
				return nil, err
			}
			insertionPoint := strings.Join([]string{sl.Category, sl.Name, f.Name, "tag"}, ".")
			if err != nil {
				return nil, err
			}
			patchs = append(patchs, &plugin.Generated{
				Content:        " " + tag,
				InsertionPoint: &insertionPoint,
			})
		}
	}
	return patchs, nil
}

func (g *Generator) getServiceDesc(scope *golang.Scope, desc Desc) (*ServiceDesc, error) {
	s := &ServiceDesc{Desc: desc}
	s.Handlers = make([]HandlerDesc, 0)

	if len(scope.Services()) == 0 {
		return nil, errors.New("service not found")
	}

	if len(scope.Services()) != 1 {
		return nil, errors.New("there should have only one service defined in a thrift file.")
	}

	s.ServiceTypeName = desc.getTypeName(scope.Services()[0].GoName().String())
	for _, f := range scope.Services()[0].Functions() {
		handler := HandlerDesc{}
		err := g.parseServiceFuncAnnotation(f.Annotations, &handler)
		if err != nil {
			return nil, err
		}

		handler.HandlerFuncName = f.GoName().String()
		handler.RequestTypeName = desc.getTypeName(f.Arguments()[0].GoTypeName().Deref().String())
		handler.ResponseTypeName = desc.getTypeName(f.ResponseGoTypeName().Deref().String())
		if handler.ResponseTypeName == "" {
			return nil, fmt.Errorf("function '%s' return type can't not be 'void'", f.Name)
		}
		s.Handlers = append(s.Handlers, handler)
	}
	return s, nil
}

func (g *Generator) genHandler(scope *golang.Scope, name string, desc Desc) ([]*plugin.Generated, error) {
	srvDesc, err := g.getServiceDesc(scope, desc)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = g.handlerTpl.Execute(&buf, srvDesc)
	if err != nil {
		return nil, err
	}
	return []*plugin.Generated{
		{
			Name:    &name,
			Content: buf.String(),
		},
	}, nil
}

func (g *Generator) genRouter(scope *golang.Scope, name string, desc Desc) ([]*plugin.Generated, error) {
	generateds := make([]*plugin.Generated, 0)
	srvDesc, err := g.getServiceDesc(scope, desc)
	if err != nil {
		return nil, err
	}

	srvDesc.Desc = desc

	finfo, err := os.Stat(name)
	if errors.Is(err, os.ErrNotExist) || finfo.IsDir() {
		writer := bytes.NewBuffer(make([]byte, 0, 1024))
		if err := g.routerTpl.Execute(writer, srvDesc); err != nil {
			return nil, err
		}
		generateds = append(generateds, &plugin.Generated{
			Name:    &name,
			Content: writer.String(),
		})
	} else {
		fb, err := ioutil.ReadFile(name)
		if err != nil {
			return nil, err
		}
		fs := string(fb)
		begin := strings.Index(fs, "// @route_gen begin")
		end := strings.Index(fs, "// @route_gen end")

		if begin == -1 || end == -1 {
			return nil, errors.New("comment '// @route_gen begin' or '// @route_gen end' not found")
		}

		generateds = append(generateds, &plugin.Generated{
			Name:    &name,
			Content: fs[:begin] + "// @route_gen begin" + plugin.InsertionPoint(srvDesc.PkgName, "Register") + fs[end:],
		})
	}

	writer := bytes.NewBuffer(make([]byte, 0, 1024))
	if err := g.routerBodyTpl.Execute(writer, map[string]interface{}{
		"Handlers":        srvDesc.Handlers,
		"ServiceTypeName": srvDesc.ServiceTypeName,
	}); err != nil {
		return nil, err
	}

	insertPoint := srvDesc.PkgName + ".Register"
	generateds = append(generateds, &plugin.Generated{
		Content:        writer.String(),
		InsertionPoint: &insertPoint,
	})
	return generateds, nil
}

func (g *Generator) genService(scope *golang.Scope, name string, desc Desc) ([]*plugin.Generated, error) {
	srvDesc, err := g.getServiceDesc(scope, desc)
	if err != nil {
		return nil, err
	}

	srvDesc.Desc = desc

	writer := bytes.NewBuffer(make([]byte, 0, 1024))
	if err := g.serviceTpl.Execute(writer, srvDesc); err != nil {
		return nil, err
	}

	return []*plugin.Generated{
		{
			Name:    &name,
			Content: writer.String(),
		},
	}, nil
}

func (g *Generator) validateOutputPath(outPath, expectPkg string) error {
	if path.IsAbs(outPath) {
		// if absolute path convert it to relative path
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if !strings.HasPrefix(outPath, cwd) {
			return fmt.Errorf("output path '%s' is not in current directory '%s'", outPath, cwd)
		}
		outPath = strings.TrimLeft(strings.TrimLeft(outPath, cwd), "/")
	}

	outPath = path.Dir(outPath)
	if outPath == "." {
		return nil
	}
	pkgName := path.Base(outPath)
	if pkgName != expectPkg {
		return fmt.Errorf("invalid package name \"%s\", expect \"%s\"", pkgName, expectPkg)
	}
	return nil
}

func (g *Generator) LoadTemplates(dir string) (handlerTpl, routerTpl, routerBodyTpl, serviceTpl *template.Template, err error) {
	handlerTpl, err = template.New("handler.tmpl").Funcs(g.tplFuncs).ParseFiles(filepath.Join(dir, "handler.tmpl"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	routerTpl, err = template.New("router.tmpl").Funcs(g.tplFuncs).ParseFiles(filepath.Join(dir, "router.tmpl"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	routerBodyTpl, err = template.New("router_body.tmpl").Funcs(g.tplFuncs).ParseFiles(filepath.Join(dir, "router_body.tmpl"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	serviceTpl, err = template.New("service.tmpl").Funcs(g.tplFuncs).ParseFiles(filepath.Join(dir, "service.tmpl"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return
}

func (g *Generator) Execute(req *plugin.Request, args *Args) (*plugin.Response, error) {
	scope, err := golang.BuildScope(g.codeutils, req.AST)
	if err != nil {
		return nil, err
	}

	pkg := g.codeutils.GetPackageName(req.AST)

	if err := g.validateOutputPath(args.HandlerPath, pkg); err != nil {
		return nil, err
	}

	if err := g.validateOutputPath(args.RouterPath, pkg); err != nil {
		return nil, err
	}

	if err := g.validateOutputPath(args.ServicePath, pkg); err != nil {
		return nil, err
	}

	g.handlerTpl, g.routerTpl, g.routerBodyTpl, g.serviceTpl, err = g.LoadTemplates(args.TemplateDir)
	if err != nil {
		return nil, err
	}

	var imports []string
	for _, include := range scope.Includes() {
		imports = append(imports, args.PackagePrefix+"/"+include.ImportPath)
	}

	desc := Desc{Version: Version, PkgName: pkg, Imports: imports}
	if args.Module != "" {
		out := strings.TrimLeft(req.OutputPath, "./")
		desc.PkgPath = fmt.Sprintf("\"%s/%s/%s\"", args.Module, out, pkg)
	} else {
		desc.PkgPath = ""
	}

	// generate router.go file patch content
	patchs, err := g.genPatchs(scope)
	if err != nil {
		return nil, err
	}
	if len(patchs) > 0 {
		outputFilePath := path.Join(req.OutputPath, g.codeutils.GetFilePath(scope.AST()))
		patchs[0].Name = &outputFilePath
		g.resp.Contents = append(g.resp.Contents, patchs...)
	}

	if args.HandlerPath != "" {
		name := args.HandlerPath
		if path.Base(args.HandlerPath) == args.HandlerPath {
			name = path.Join(req.OutputPath, pkg, args.HandlerPath)
		}
		handlers, err := g.genHandler(scope, name, desc)
		if err != nil {
			return nil, err
		}
		if len(handlers) > 0 {
			g.resp.Contents = append(g.resp.Contents, handlers...)
		}
	}

	if args.RouterPath != "" {
		name := args.RouterPath
		if path.Base(args.RouterPath) == args.RouterPath {
			name = path.Join(req.OutputPath, pkg, args.RouterPath)
		}
		routers, err := g.genRouter(scope, name, desc)
		if err != nil {
			return nil, err
		}
		if len(routers) > 0 {
			g.resp.Contents = append(g.resp.Contents, routers...)
		}
	}

	if args.ServicePath != "" {
		name := args.ServicePath
		if path.Base(args.ServicePath) == args.ServicePath {
			name = path.Join(req.OutputPath, pkg, args.ServicePath)
		}
		routers, err := g.genService(scope, name, desc)
		if err != nil {
			return nil, err
		}
		if len(routers) > 0 {
			g.resp.Contents = append(g.resp.Contents, routers...)
		}
	}

	g.resp.Warnings = g.warns
	return g.resp, nil
}
