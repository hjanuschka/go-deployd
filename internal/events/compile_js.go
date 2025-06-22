package events

import (
	v8 "rogchap.com/v8go"
)

// CompileJS compiles JavaScript source to an UnboundScript
func CompileJS(filename, source string) (*v8.UnboundScript, error) {
	isolate := v8.NewIsolate()
	defer isolate.Dispose()
	
	unbound, err := isolate.CompileUnboundScript(source, filename, v8.CompileOptions{})
	if err != nil {
		return nil, err
	}
	return unbound, nil
}