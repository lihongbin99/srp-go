package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// readIniFile 从文件读取配置
// filePath: 配置文件的路径
// fileConfig: 配置存储对象
// err: 读取时异常
func readIniFile(filePath string, fileConfig Config) (err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}

	section := ""
	var parseErr error = nil
	buf := bufio.NewReader(file)
	var text string
	for {
		// 逐行读取配置文件
		text, err = buf.ReadString('\n')

		// 去空格
		text = strings.TrimSpace(text)
		if text != "" {
			// 解析配置文件并设置进Config对象
			if section, parseErr = parseInI(section, strings.TrimSpace(text), fileConfig); parseErr != nil {
				err = parseErr
				return
			}
		}

		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
	}
}

func parseInI(section, text string, fileConfig Config) (string, error) {
	length := len(text)
	if length <= 0 || text[0] == ';' { // 注释
		return section, nil
	}

	// section
	if text[0] == '[' && text[length-1] == ']' {
		section = text[1 : length-1]
		return section, nil
	}

	// 获取名称和值
	index := strings.Index(text, "=")
	if index < 0 {
		return section, fmt.Errorf("ini config no find '=' in %s", text)
	} else if index == 0 {
		return section, fmt.Errorf("ini config no key name in %s", text)
	}
	// 去空格
	name := strings.TrimSpace(text[0:index])
	parameters := strings.TrimSpace(text[index+1 : length])

	return section, setConfigValue(section, name, parameters, fileConfig)
}

func setConfigValue(section string, name string, parameters string, fileConfig Config) error {
	if parameters == "" {
		return nil
	}

	rType := reflect.TypeOf(fileConfig)
	rValue := reflect.ValueOf(fileConfig)

	// 获取 Section Config 对象
	for i := 0; i < rType.NumField(); i++ {
		field := rType.Field(i)
		if strings.HasPrefix(section, field.Name) {
			configObject := rValue.Field(i).Interface()

			// 获取 Name Config 对象
			names := strings.Split(name, ".")
			for j := 0; j < len(names)-1; j++ {
				configObjectType := reflect.TypeOf(configObject).Elem()
				configObjectValue := reflect.ValueOf(configObject).Elem()
				for i := 0; i < configObjectType.NumField(); i++ {
					field := configObjectType.Field(i)
					if names[j] == field.Name {
						configObject = configObjectValue.Field(i).Interface()
						break
					}
				}
			}

			name = names[len(names)-1]
			// 获取字段
			configObjectType := reflect.TypeOf(configObject).Elem()
			configObjectValue := reflect.ValueOf(configObject)

			// 检测类型 struct or map
			if configObjectValue.Type().Kind() == reflect.Map {
				keys := configObjectValue.MapKeys()
				for _, k := range keys {
					if k.Interface() == section {
						serviceConfig := configObjectValue.MapIndex(k).Interface()
						// 设置配置值
						err := setMapValue(serviceConfig, name, parameters)
						return err
					}
				}
				// 创建配置对象
				serviceConfig := reflect.New(configObjectType.Elem()).Interface()
				// 设置配置值
				if err := setMapValue(serviceConfig, name, parameters); err != nil {
					return err
				}
				// 添加进配置map
				configObjectValue.SetMapIndex(reflect.ValueOf(section), reflect.ValueOf(serviceConfig))
			} else if configObjectValue.Elem().Type().Kind() == reflect.Struct {
				for i := 0; i < configObjectType.NumField(); i++ {
					field := configObjectType.Field(i)
					if name == field.Name {
						// 设置字段
						value := configObjectValue.Elem().Field(i)
						if err := setValue(value, parameters); err != nil {
							return err
						}
						break
					}
				}
			}
		}
	}
	return nil
}

func setMapValue(serviceConfig interface{}, name, parameters string) error {
	serviceConfigType := reflect.TypeOf(serviceConfig)
	serviceConfigValue := reflect.ValueOf(serviceConfig)
	for i := 0; i < serviceConfigType.Elem().NumField(); i++ {
		field := serviceConfigType.Elem().Field(i)
		if name == field.Name {
			// 设置字段
			value := serviceConfigValue.Elem().Field(i)
			if err := setValue(value, parameters); err != nil {
				return err
			}
		}
	}
	return nil
}

func setValue(value reflect.Value, parameters string) error {
	switch value.Type().Kind() {
	case reflect.String:
		value.SetString(parameters)
	case reflect.Int:
		pi, err := strconv.Atoi(parameters)
		if err != nil {
			return err
		}
		value.SetInt(int64(pi))
	case reflect.Bool:
		value.SetBool(parameters == "true")
	}
	return nil
}
