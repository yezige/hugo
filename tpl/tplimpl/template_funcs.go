// Copyright 2017-present The Hugo Authors. All rights reserved.
//
// Portions Copyright The Go Authors.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tplimpl

import (
	"context"
	"reflect"
	"strings"

	"github.com/yezige/hugo/common/hreflect"
	"github.com/yezige/hugo/common/maps"
	"github.com/yezige/hugo/tpl"

	template "github.com/yezige/hugo/tpl/internal/go_templates/htmltemplate"
	texttemplate "github.com/yezige/hugo/tpl/internal/go_templates/texttemplate"

	"github.com/yezige/hugo/deps"

	"github.com/yezige/hugo/tpl/internal"

	// Init the namespaces
	_ "github.com/yezige/hugo/tpl/cast"
	_ "github.com/yezige/hugo/tpl/collections"
	_ "github.com/yezige/hugo/tpl/compare"
	_ "github.com/yezige/hugo/tpl/crypto"
	_ "github.com/yezige/hugo/tpl/css"
	_ "github.com/yezige/hugo/tpl/data"
	_ "github.com/yezige/hugo/tpl/debug"
	_ "github.com/yezige/hugo/tpl/diagrams"
	_ "github.com/yezige/hugo/tpl/encoding"
	_ "github.com/yezige/hugo/tpl/fmt"
	_ "github.com/yezige/hugo/tpl/hugo"
	_ "github.com/yezige/hugo/tpl/images"
	_ "github.com/yezige/hugo/tpl/inflect"
	_ "github.com/yezige/hugo/tpl/js"
	_ "github.com/yezige/hugo/tpl/lang"
	_ "github.com/yezige/hugo/tpl/math"
	_ "github.com/yezige/hugo/tpl/openapi/openapi3"
	_ "github.com/yezige/hugo/tpl/os"
	_ "github.com/yezige/hugo/tpl/page"
	_ "github.com/yezige/hugo/tpl/partials"
	_ "github.com/yezige/hugo/tpl/path"
	_ "github.com/yezige/hugo/tpl/reflect"
	_ "github.com/yezige/hugo/tpl/resources"
	_ "github.com/yezige/hugo/tpl/safe"
	_ "github.com/yezige/hugo/tpl/site"
	_ "github.com/yezige/hugo/tpl/strings"
	_ "github.com/yezige/hugo/tpl/templates"
	_ "github.com/yezige/hugo/tpl/time"
	_ "github.com/yezige/hugo/tpl/transform"
	_ "github.com/yezige/hugo/tpl/urls"
)

var (
	_                texttemplate.ExecHelper = (*templateExecHelper)(nil)
	zero             reflect.Value
	contextInterface = reflect.TypeOf((*context.Context)(nil)).Elem()
)

type templateExecHelper struct {
	running    bool // whether we're in server mode.
	site       reflect.Value
	siteParams reflect.Value
	funcs      map[string]reflect.Value
}

func (t *templateExecHelper) GetFunc(ctx context.Context, tmpl texttemplate.Preparer, name string) (fn reflect.Value, firstArg reflect.Value, found bool) {
	if fn, found := t.funcs[name]; found {
		if fn.Type().NumIn() > 0 {
			first := fn.Type().In(0)
			if first.Implements(contextInterface) {
				// TODO(bep) check if we can void this conversion every time -- and if that matters.
				// The first argument may be context.Context. This is never provided by the end user, but it's used to pass down
				// contextual information, e.g. the top level data context (e.g. Page).
				return fn, reflect.ValueOf(ctx), true
			}
		}

		return fn, zero, true
	}
	return zero, zero, false
}

func (t *templateExecHelper) Init(ctx context.Context, tmpl texttemplate.Preparer) {
}

func (t *templateExecHelper) GetMapValue(ctx context.Context, tmpl texttemplate.Preparer, receiver, key reflect.Value) (reflect.Value, bool) {
	if params, ok := receiver.Interface().(maps.Params); ok {
		// Case insensitive.
		keystr := strings.ToLower(key.String())
		v, found := params[keystr]
		if !found {
			return zero, false
		}
		return reflect.ValueOf(v), true
	}

	v := receiver.MapIndex(key)

	return v, v.IsValid()
}

var typeParams = reflect.TypeOf(maps.Params{})

func (t *templateExecHelper) GetMethod(ctx context.Context, tmpl texttemplate.Preparer, receiver reflect.Value, name string) (method reflect.Value, firstArg reflect.Value) {
	if t.running {
		switch name {
		case "GetPage", "Render":
			if info, ok := tmpl.(tpl.Info); ok {
				if m := receiver.MethodByName(name + "WithTemplateInfo"); m.IsValid() {
					return m, reflect.ValueOf(info)
				}
			}
		}
	}

	if strings.EqualFold(name, "mainsections") && receiver.Type() == typeParams && receiver.Pointer() == t.siteParams.Pointer() {
		// MOved to site.MainSections in Hugo 0.112.0.
		receiver = t.site
		name = "MainSections"

	}

	fn := hreflect.GetMethodByName(receiver, name)
	if !fn.IsValid() {
		return zero, zero
	}

	if fn.Type().NumIn() > 0 {
		first := fn.Type().In(0)
		if first.Implements(contextInterface) {
			// The first argument may be context.Context. This is never provided by the end user, but it's used to pass down
			// contextual information, e.g. the top level data context (e.g. Page).
			return fn, reflect.ValueOf(ctx)
		}
	}

	return fn, zero
}

func newTemplateExecuter(d *deps.Deps) (texttemplate.Executer, map[string]reflect.Value) {
	funcs := createFuncMap(d)
	funcsv := make(map[string]reflect.Value)

	for k, v := range funcs {
		vv := reflect.ValueOf(v)
		funcsv[k] = vv
	}

	// Duplicate Go's internal funcs here for faster lookups.
	for k, v := range template.GoFuncs {
		if _, exists := funcsv[k]; !exists {
			vv, ok := v.(reflect.Value)
			if !ok {
				vv = reflect.ValueOf(v)
			}
			funcsv[k] = vv
		}
	}

	for k, v := range texttemplate.GoFuncs {
		if _, exists := funcsv[k]; !exists {
			funcsv[k] = v
		}
	}

	exeHelper := &templateExecHelper{
		running:    d.Conf.Running(),
		funcs:      funcsv,
		site:       reflect.ValueOf(d.Site),
		siteParams: reflect.ValueOf(d.Site.Params()),
	}

	return texttemplate.NewExecuter(
		exeHelper,
	), funcsv
}

func createFuncMap(d *deps.Deps) map[string]any {
	funcMap := template.FuncMap{}

	// Merge the namespace funcs
	for _, nsf := range internal.TemplateFuncsNamespaceRegistry {
		ns := nsf(d)
		if _, exists := funcMap[ns.Name]; exists {
			panic(ns.Name + " is a duplicate template func")
		}
		funcMap[ns.Name] = ns.Context
		for _, mm := range ns.MethodMappings {
			for _, alias := range mm.Aliases {
				if _, exists := funcMap[alias]; exists {
					panic(alias + " is a duplicate template func")
				}
				funcMap[alias] = mm.Method
			}
		}
	}

	if d.OverloadedTemplateFuncs != nil {
		for k, v := range d.OverloadedTemplateFuncs {
			funcMap[k] = v
		}
	}

	return funcMap
}
