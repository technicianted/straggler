// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package cmd

import (
	"fmt"
	"reflect"
	"time"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	_ "embed"
)

var (
	//go:embed usage.tmpl
	usageTemplate string
)

func EnrichCommand(cmd *cobra.Command, flagsStruct interface{}) {
	cmd.SetUsageTemplate(usageTemplate)
	if flagsStruct != nil {
		addCommandFlags(flagsStruct, cmd.Flags())
	}
}

func addCommandFlags(flagsStruct interface{}, flagSet *flag.FlagSet) {
	doAddCommandFlags(reflect.ValueOf(flagsStruct).Elem(), flagSet)
}

func doAddCommandFlags(val reflect.Value, flagSet *flag.FlagSet) {
	valT := val.Type()
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		value := valueField.Interface()
		typeField := valT.Field(i)
		if valueField.Kind() == reflect.Struct {
			doAddCommandFlags(valueField, flagSet)
			continue
		}

		cliArgName := typeField.Tag.Get("cliArgName")
		if cliArgName == "" {
			continue
		}
		cliArgDescription := typeField.Tag.Get("cliArgDescription")
		valueFieldPointer := valueField.Addr().Interface()

		switch v := value.(type) {
		case string:
			flagSet.StringVar(valueFieldPointer.(*string), cliArgName, v, cliArgDescription)
		case *string:
			flagSet.StringVar(v, cliArgName, *v, cliArgDescription)
		case int:
			flagSet.IntVar(valueFieldPointer.(*int), cliArgName, v, cliArgDescription)
		case *int:
			flagSet.IntVar(v, cliArgName, *v, cliArgDescription)
		case bool:
			flagSet.BoolVar(valueFieldPointer.(*bool), cliArgName, v, cliArgDescription)
		case *bool:
			flagSet.BoolVar(v, cliArgName, *v, cliArgDescription)
		case time.Duration:
			flagSet.DurationVar(valueFieldPointer.(*time.Duration), cliArgName, v, cliArgDescription)
		case *time.Duration:
			flagSet.DurationVar(v, cliArgName, *v, cliArgDescription)
		case []string:
			flagSet.StringSliceVar(valueFieldPointer.(*[]string), cliArgName, v, cliArgDescription)
		default:
			panic(fmt.Sprintf("unsupported type: %T", v))
		}
		group := typeField.Tag.Get("cliArgGroup")
		flagSet.SetGroup(cliArgName, group)
	}
}
