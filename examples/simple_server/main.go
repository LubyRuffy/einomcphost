package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"

	mcpp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Input struct {
	A int `json:"a" mcp:"required;default:1;max:100;min:0;enum:1,2,3;maxLength:10;minLength:1"`
	B int `json:"b" mcp:"required;default:2;max:100;min:0;enum:1,2,3;maxLength:10;minLength:1"`
}

func Sum1(i Input) (int, error) {
	return i.A + i.B, nil
}

func Sum2(ctx context.Context, i *Input) (int, error) {
	return i.A + i.B, nil
}

func getParamType(f any) (reflect.Value, reflect.Type, bool, int, error) {
	// refelct判断是否函数
	fn := reflect.ValueOf(f)
	if fn.Kind() != reflect.Func {
		return reflect.Value{}, nil, false, 0, fmt.Errorf("f is not a function")
	}

	// 获取函数参数
	fnType := fn.Type()
	numIn := fnType.NumIn()
	if numIn < 1 || numIn > 2 {
		return reflect.Value{}, nil, false, 0, fmt.Errorf("function must have 1 or 2 arguments")
	}

	// 如果参数为2，则需要判断第一个参数是否为 context.Context
	if numIn == 2 {
		if fnType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
			return reflect.Value{}, nil, false, 0, fmt.Errorf("function must have a context.Context as the first argument")
		}
	}

	// 解析真实的参数类型
	realInputType := fnType.In(numIn - 1)
	isPtr := realInputType.Kind() == reflect.Ptr
	var actualType reflect.Type
	if isPtr {
		actualType = realInputType.Elem()
	} else {
		actualType = realInputType
	}

	return fn, actualType, isPtr, numIn, nil
}

// TransportSimpleFunction 将函数转换为 server.ToolHandlerFunc
// 输入参数为 f func(xxx)，输出为 server.ToolHandlerFunc
// 支持两个函数格式
// 1. func(input any) (any, error) // input 可以是结构体或者结构体指针
// 2. func(ctx context.Context, input any) (any, error) // input 可以是结构体或者结构体指针
func TransportSimpleFunction(f any) (server.ToolHandlerFunc, error) {
	fn, actualType, isPtr, numIn, err := getParamType(f)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, request mcpp.CallToolRequest) (*mcpp.CallToolResult, error) {
		// 动态构建函数参数realInputType的类型值
		realInputPtr := reflect.New(actualType)
		if err := request.BindArguments(realInputPtr.Interface()); err != nil {
			return mcpp.NewToolResultErrorFromErr("bind arguments failed", err), nil
		}

		// 获取真实的参数值
		var realInputVal reflect.Value
		if isPtr {
			realInputVal = realInputPtr
		} else {
			realInputVal = realInputPtr.Elem()
		}

		// 动态调用函数
		var callArgs []reflect.Value
		if numIn == 1 {
			callArgs = []reflect.Value{realInputVal}
		} else {
			callArgs = []reflect.Value{reflect.ValueOf(ctx), realInputVal}
		}

		results := fn.Call(callArgs)
		output := results[0].Interface()
		var err error
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}

		if err != nil {
			return mcpp.NewToolResultErrorFromErr("call function failed", err), nil
		}
		outputJson, err := json.Marshal(output)
		if err != nil {
			return mcpp.NewToolResultErrorFromErr("marshal output failed", err), nil
		}

		// 将结果转换为mcpp.CallToolResult
		return &mcpp.CallToolResult{Content: []mcpp.Content{
			mcpp.NewTextContent(string(outputJson)),
		}}, nil

	}, nil
}

// MustTransportSimpleFunction 将函数转换为 server.ToolHandlerFunc，如果转换失败，则panic
func MustTransportSimpleFunction(f any) server.ToolHandlerFunc {
	tool, err := TransportSimpleFunction(f)
	if err != nil {
		log.Fatalf("Failed to transport function: %v", err)
	}
	return tool
}

func MustTransportSimpleTool(name string, description string, f any) mcpp.Tool {
	_, toolType, _, _, err := getParamType(f)
	if err != nil {
		log.Fatalf("Failed to get param type: %v", err)
	}

	fieldOptions := []mcpp.ToolOption{mcpp.WithDescription(description)}
	fmt.Println("toolType", toolType, toolType.Kind(), toolType.Name(), toolType.Kind().String())
	if toolType.Kind() == reflect.Struct {
		for i := 0; i < toolType.NumField(); i++ {
			field := toolType.Field(i)

			var tagOptions []mcpp.PropertyOption
			if field.Tag.Get("mcp") != "" {
				tags := strings.Split(field.Tag.Get("mcp"), ";")
				for _, tag := range tags {
					k, v, _ := strings.Cut(tag, ":")
					switch k {
					case "required":
						tagOptions = append(tagOptions, mcpp.Required())
					case "default":
						switch field.Type.Kind() {
						case reflect.Int:
							num, err := strconv.ParseInt(v, 10, 64)
							if err != nil {
								log.Fatalf("Failed to parse default value for field %s: %v", field.Name, err)
							}
							tagOptions = append(tagOptions, mcpp.DefaultNumber(float64(num)))
						case reflect.Float64:
							num, err := strconv.ParseFloat(v, 64)
							if err != nil {
								log.Fatalf("Failed to parse default value for field %s: %v", field.Name, err)
							}
							tagOptions = append(tagOptions, mcpp.DefaultNumber(num))
						case reflect.Float32:
							num, err := strconv.ParseFloat(v, 32)
							if err != nil {
								log.Fatalf("Failed to parse default value for field %s: %v", field.Name, err)
							}
							tagOptions = append(tagOptions, mcpp.DefaultNumber(float64(num)))
						case reflect.String:
							tagOptions = append(tagOptions, mcpp.DefaultString(v))
						case reflect.Bool:
							tagOptions = append(tagOptions, mcpp.DefaultBool(v == "true"))
						default:
							log.Fatalf("Unsupported type for field %s: %v", field.Name, field.Type.Kind())
						}
					case "max":
						num, err := strconv.ParseFloat(v, 64)
						if err != nil {
							log.Fatalf("Failed to parse max value for field %s: %v", field.Name, err)
						}
						tagOptions = append(tagOptions, mcpp.Max(num))
					case "min":
						num, err := strconv.ParseFloat(v, 64)
						if err != nil {
							log.Fatalf("Failed to parse min value for field %s: %v", field.Name, err)
						}
						tagOptions = append(tagOptions, mcpp.Min(num))
					case "enum":
						tagOptions = append(tagOptions, mcpp.Enum(strings.Split(v, ",")...))
					case "maxLength":
						num, err := strconv.Atoi(v)
						if err != nil {
							log.Fatalf("Failed to parse maxLength value for field %s: %v", field.Name, err)
						}
						tagOptions = append(tagOptions, mcpp.MaxLength(num))
					case "minLength":
						num, err := strconv.Atoi(v)
						if err != nil {
							log.Fatalf("Failed to parse minLength value for field %s: %v", field.Name, err)
						}
						tagOptions = append(tagOptions, mcpp.MinLength(num))
					}
				}

				switch toolType.Field(i).Type.Kind() {
				case reflect.Int:
					fieldOptions = append(fieldOptions, mcpp.WithNumber(toolType.Field(i).Name, tagOptions...))
				case reflect.String:
					fieldOptions = append(fieldOptions, mcpp.WithString(toolType.Field(i).Name, tagOptions...))
				case reflect.Bool:
					fieldOptions = append(fieldOptions, mcpp.WithBoolean(toolType.Field(i).Name, tagOptions...))
				default:
					log.Fatalf("Unsupported type for field %s: %v", field.Name, field.Type.Kind())
				}
			}
		}
	}

	// 解析结构体里面的mcp标签

	return mcpp.NewTool(name, fieldOptions...)
}

func MustAddTool(s *server.MCPServer, name string, description string, f any) {
	s.AddTool(MustTransportSimpleTool(name, description, f), MustTransportSimpleFunction(f))
}

func main() {
	s := server.NewMCPServer("test", "v1")

	MustAddTool(s, "sum1", "sum two numbers", Sum1)
	MustAddTool(s, "sum2", "sum two numbers", Sum2)

	srv := server.NewStreamableHTTPServer(s)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	go func() {
		log.Println("Starting server on port 28080")
		if err := srv.Start(":28080"); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Println("Server started: http://localhost:28080/mcp")

	<-signalChan
	srv.Shutdown(context.Background())
}
