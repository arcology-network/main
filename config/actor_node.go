package config

import "fmt"

type ActorNode struct {
	Name   string
	Params map[string]interface{}
	Subs   map[string]*ActorNode
}

func ParseActors(cfg map[string]map[string]interface{}) map[string]*ActorNode {
	result := make(map[string]*ActorNode)
	for name, raw := range cfg {
		result[name] = parseActorNode(name, raw)
	}
	return result
}

func parseActorNode(name string, raw map[string]interface{}) *ActorNode {
	node := &ActorNode{
		Name:   name,
		Params: make(map[string]interface{}),
		Subs:   make(map[string]*ActorNode),
	}

	for k, v := range raw {
		if m, ok := asStringMap(v); ok {
			node.Subs[k] = parseActorNode(k, m)
		} else {
			node.Params[k] = v
		}
	}

	return node
}

func normalizeMap(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for k, v := range x {
			ks := fmt.Sprintf("%v", k)
			m[ks] = normalizeMap(v)
		}
		return m
	case []interface{}:
		for i, v := range x {
			x[i] = normalizeMap(v)
		}
		return x
	default:
		return i
	}
}
func asStringMap(v interface{}) (map[string]interface{}, bool) {
	switch m := v.(type) {
	case map[string]interface{}:
		return m, true
	case map[interface{}]interface{}:
		r := make(map[string]interface{})
		for k, v := range m {
			r[fmt.Sprintf("%v", k)] = v
		}
		return r, true
	default:
		return nil, false
	}
}
