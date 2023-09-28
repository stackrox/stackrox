package main

import (
	"bytes"
	"time"

	// Embed is used to import the template files
	_ "embed"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/fatih/camelcase"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/tools/imports"
)

const (
	storageProtoDir = "./proto/storage"
	apiProtoDir     = "./proto/api"
	v1Prefix        = "v1"
	v2Prefix        = "v2"

	outputRoot = "./roxctl/generated"
)

var (
	// Store map of HTTP methods to roxctl commands.
	httpMethodToRoxctlVerb = map[string]string{
		"get":    "get",
		"post":   "create",
		"put":    "update",
		"delete": "delete",
	}

	methodPrefixes = set.NewStringSet("get", "create", "add", "post", "update", "put", "patch", "remove", "delete", "cancel")
)

//go:embed get.go.tpl
var getCmdFile string

//go:embed get_sub_cmd.go.tpl
var getSubCmdFile string

var (
	getCmdTemplate    = newTemplate(getCmdFile)
	getSubCmdTemplate = newTemplate(getSubCmdFile)
)

type serviceProps struct {
	Name    string
	Package string
	Prefix  string
	Methods []*methodProps
}

type methodProps struct {
	GRPC *GRPCMethodInfo
	HTTP *HTTPInfo
}

type GRPCMethodInfo struct {
	Name     string
	Resource string
	Input    Field
	Output   Field
}

type HTTPInfo struct {
	Method string
	Path   string
	Params []ParamInfo
}

type ParamInfo struct {
	Name string
	// command flag name. e.g. flag name for "id" in /clusters/{id} is "cluster".
	CmdFlagName string
}

type SubCmd struct {
	Name string
	Dir  string
}

type Field struct {
	// does not apply to method args and return args.
	GoName string
	// does not apply to method args and return args.
	ProtoBufName string
	DataType     string
	paramInfo    ParamInfo
}

func main() {
	c := &cobra.Command{
		Use: "generate roxctl command",
	}

	var protoFile string
	c.Flags().StringVar(&protoFile, "proto-file", "", ".proto file path of the service")
	utils.Must(c.MarkFlagRequired("proto-file"))

	c.RunE = func(*cobra.Command, []string) error {
		fileName := path.Join(apiProtoDir, protoFile)
		_, err := protoparse.ResolveFilenames([]string{apiProtoDir}, fileName)
		if err != nil {
			log.Fatalf(".proto file %s could not be resolved: %v", protoFile, err)
		}

		service := parseProtoFile(fileName)

		for _, method := range service.Methods {
			// TODO: Handle other methods.
			if method.HTTP.Method != "get" {
				continue
			}
			if len(method.HTTP.Params) == 0 || method.HTTP.Params[len(method.HTTP.Params)-1].Name != "id" {
				continue
			}

			fmt.Println(readable.Time(time.Now()), "Generating for", method.GRPC.Name)

			templateMap := map[string]interface{}{
				"service":     service,
				"methodProps": method,
			}
			dir := commandFilePath(method)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			cmdFilePath := path.Join(dir, "command.go")

			switch method.HTTP.Method {
			case "get":
				err = renderFile(templateMap, getSubCmdTemplate, cmdFilePath)
			default:
				return errors.Errorf("HTTP %s method is unsupported", method.HTTP.Method)
			}
			if err != nil {
				return err
			}
		}
		if err := reconcileGetCmd(path.Join(outputRoot, "get")); err != nil {
			return errors.Wrap(err, "failed to reconcile get command")
		}
		return nil
	}

	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func parseProtoFile(fileName string) *serviceProps {
	parser := protoparse.Parser{}
	fieldDesc, err := parser.ParseFilesButDoNotLink(fileName)
	if err != nil {
		log.Fatalf("unable to parse .proto file %s: %v", fileName, err)
		return nil
	}

	var service *serviceProps
	for _, desc := range fieldDesc {
		svcs := desc.GetService()
		for _, svc := range svcs {
			service = &serviceProps{
				Name:    svc.GetName(),
				Package: desc.GetPackage(),
				Prefix:  strings.TrimSuffix(svc.GetName(), "Service"),
			}

			var methods []*methodProps
			for _, methodDesc := range svc.GetMethod() {
				method := &methodProps{
					GRPC: &GRPCMethodInfo{
						Name:     methodDesc.GetName(),
						Resource: getResource(methodDesc.GetName()),
						Input: Field{
							DataType: parseInOutType(service.Package, methodDesc.GetInputType()),
						},
						Output: Field{
							DataType: parseInOutType(service.Package, methodDesc.GetOutputType()),
						},
					},
				}

				for _, methodOption := range methodDesc.GetOptions().GetUninterpretedOption() {
					var validHTTPMethod bool
					parts := strings.Split(methodOption.GetAggregateValue(), " ")
					for _, part := range parts {
						part = strings.Trim(part, " :\"")
						if validHTTPMethod {
							if method.HTTP == nil {
								method.HTTP = &HTTPInfo{}
							}
							method.HTTP.Path = part
							methods = append(methods, method)
							break
						}

						_, validHTTPMethod = httpMethodToRoxctlVerb[part]
						if validHTTPMethod {
							method.HTTP = &HTTPInfo{
								Method: part,
							}
						}
					}
					if validHTTPMethod {
						populateParams(method)
						break
					}
				}
			}
			service.Methods = append(service.Methods, methods...)
		}
	}

	return service
}

func parseInOutType(servicePkg, typ string) string {
	if strings.Contains(typ, ".") {
		return typ
	}
	return servicePkg + "." + typ
}

func populateParams(props *methodProps) {
	if props.HTTP == nil || props.HTTP.Path == "" {
		log.Fatalf("HTTP info not found")
	}

	parts := strings.Split(props.HTTP.Path, "/")

	var params []ParamInfo
	var prev string
	for _, part := range parts {
		l := strings.Index(part, "{")
		r := strings.Index(part, "}")
		if l == -1 || r == -1 {
			prev = part
			continue
		}

		param := part[l+1 : r]
		flagName := param
		if param == "id" {
			flagName = prev
		}
		params = append(params, ParamInfo{Name: param, CmdFlagName: flagName})
	}
	props.HTTP.Params = params
}

func renderFile(templateMap map[string]interface{}, temp func(s string) *template.Template, templateFileName string) error {
	buf := bytes.NewBuffer(nil)
	if err := temp(templateFileName).Execute(buf, templateMap); err != nil {
		return err
	}
	file := buf.Bytes()

	formatted, err := imports.Process(templateFileName, file, nil)
	if err != nil {
		return err
	}
	if err := os.WriteFile(templateFileName, formatted, 0644); err != nil {
		return err
	}
	return nil
}

func newTemplate(tpl string) func(name string) *template.Template {
	return func(name string) *template.Template {
		return template.Must(template.New(name).Option("missingkey=error").Parse(tpl))
	}
}

func reconcileGetCmd(dir string) error {
	fmt.Println(readable.Time(time.Now()), "Reconciling 'get' command")

	entries, err := os.ReadDir(dir)
	if err != nil {
		return errors.Errorf("failed to read directory %s", dir)
	}

	var subCmds []SubCmd
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subCmds = append(subCmds, SubCmd{
			Name: entry.Name(),
			Dir:  path.Join(dir, entry.Name()),
		})
	}

	templateMap := map[string]interface{}{
		"subCmds": subCmds,
	}
	return renderFile(templateMap, getCmdTemplate, path.Join(dir, "command.go"))
}

func commandFilePath(method *methodProps) string {
	if method == nil {
		log.Fatal("method is nil")
	}

	roxctlVerb, validHTTPMethod := httpMethodToRoxctlVerb[method.HTTP.Method]
	if !validHTTPMethod {
		log.Fatalf("HTTP method %s unsupported", method.HTTP.Method)
	}

	subDir := strings.ToLower(method.GRPC.Name)
	for _, prefix := range methodPrefixes.AsSlice() {
		if strings.HasPrefix(subDir, prefix) {
			subDir = strings.TrimPrefix(subDir, prefix)
			break
		}
	}

	pathComponents := []string{
		outputRoot,
		roxctlVerb,
		subDir,
	}
	return path.Join(pathComponents...)
}

func getResource(name string) string {
	parts := camelcase.Split(name)
	if len(parts) < 1 {
		return strings.ToLower(name)
	}
	for idx := 0; idx < len(parts); idx++ {
		part := &parts[idx]
		*part = strings.ToLower(parts[idx])
	}
	if methodPrefixes.Contains(parts[0]) {
		return strings.Join(parts[1:], "")
	}
	return strings.ToLower(name)
}
