package cache

import (
	"strings"
	"sync"
)

var _ Cache[interface{}] = &PrefixCache[interface{}]{}

const DefaultKeySeparator string = "/"
const rootNodeKey = "root"

type PrefixCache[I interface{}] struct {
	separator string
	rootNode  *node[I]
	mu        sync.RWMutex
}

func NewPrefixCache[I interface{}]() *PrefixCache[I] {
	return &PrefixCache[I]{
		separator: DefaultKeySeparator,
		rootNode:  nil,
		mu:        sync.RWMutex{},
	}
}

func (p *PrefixCache[I]) Get(key Key) (I, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	keyParts := p.splitKey(key)
	if p.rootNode == nil {
		return *new(I), false
	}
	currentNode := p.rootNode
	var path []string
	for _, part := range keyParts {
		path = append(path, string(part))
		if child, ok := currentNode.children[part]; ok {
			currentNode = child
		} else {
			return *new(I), false
		}
	}
	if currentNode.value == nil {
		return *new(I), false
	}
	return *currentNode.value, true
}

func (p *PrefixCache[I]) Set(key Key, value I) {
	p.mu.Lock()
	defer p.mu.Unlock()
	keyParts := p.splitKey(key)
	if p.rootNode == nil {
		p.rootNode = newNode[I](rootNodeKey)
	}
	currentNode := p.rootNode
	var path []string
	for _, part := range keyParts {
		path = append(path, string(part))
		if child, ok := currentNode.children[part]; ok {
			currentNode = child
		} else {
			child = newNode[I](Key(strings.Join(path, p.separator)))
			currentNode.children[part] = child
			currentNode = child
		}
	}
	currentNode.value = &value
}

func (p *PrefixCache[I]) Delete(key Key) {
	p.mu.Lock()
	defer p.mu.Unlock()
	nodes := p.nodesInPath(key)
	if nodes == nil {
		// path not found
		return
	}

	// nil out stored value
	nodes[len(nodes)-1].value = nil

	// walk backwards deleting the nodes
	for i := len(nodes) - 2; i >= 0; i-- {
		node := nodes[i]
		child := nodes[i+1]
		if len(child.children) == 0 && child.value == nil {
			delete(node.children, child.key)
		} else {
			break
		}
	}
}

func (p *PrefixCache[I]) DeleteByPrefix(prefix Key) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deleteByPrefixRecursive(p.rootNode, p.splitKey(prefix), 0)
}

func (p *PrefixCache[I]) deleteByPrefixRecursive(node *node[I], parts []Key, index int) {
	if len(parts) == index {
		// delete all keys in subtree
		for key := range node.children {
			p.deleteByPrefixRecursive(node.children[key], []Key{key}, 0)
		}
		node.children = newMap[I]()
		node.value = nil
	} else {
		// follow path corresponding to prefix
		part := parts[index]
		if part == "*" {
			// wildcard, delete all keys in subtree
			for key := range node.children {
				p.deleteByPrefixRecursive(node.children[key], parts, index+1)

				// delete child if it's a leaf node
				if len(node.children[key].children) == 0 && node.children[key].value == nil {
					delete(node.children, key)
				}
			}
		} else {
			if child, ok := node.children[part]; ok {
				p.deleteByPrefixRecursive(child, parts, index+1)

				// delete child if it's a leaf node
				if len(child.children) == 0 && child.value == nil {
					delete(node.children, part)
				}
			}
		}
	}
}

func (p *PrefixCache[I]) Iterate(f func(key Key, value I) error) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// depth first search
	var stack []*node[I]

	// prime stack with first level of nodes
	for _, child := range p.rootNode.children {
		stack = append(stack, child)
	}

	for len(stack) > 0 {
		// pop
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if len(node.children) == 0 && node.value != nil {
			if err := f(node.key, *node.value); err != nil {
				return err
			}
		} else {
			// push children
			for _, child := range node.children {
				stack = append(stack, child)
			}
		}
	}
	return nil
}

func (p *PrefixCache[I]) PrefixScan(prefix string, fn func(key Key, value I) error) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	parts := strings.Split(prefix, "/")
	return p.prefixScanRecursive(p.rootNode, parts, fn)
}

func (p *PrefixCache[I]) prefixScanRecursive(node *node[I], parts []string, fn func(key Key, value I) error) error {
	if len(parts) == 0 {
		// reached end of prefix, collect all values in subtree
		if len(node.children) == 0 && node.value != nil {
			if err := fn(node.key, *node.value); err != nil {
				return err
			}
		} else {
			for _, child := range node.children {
				if err := p.prefixScanRecursive(child, parts, fn); err != nil {
					return err
				}
			}
		}
	} else {
		// follow path corresponding to prefix
		part := Key(parts[0])
		if part == "*" {
			// wildcard, explore all child nodes
			for _, child := range node.children {
				if err := p.prefixScanRecursive(child, parts[1:], fn); err != nil {
					return err
				}
			}
		} else {
			if child, ok := node.children[part]; ok {
				if err := p.prefixScanRecursive(child, parts[1:], fn); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (p *PrefixCache[I]) splitKey(key Key) []Key {
	parts := strings.Split(string(key), p.separator)
	keyParts := make([]Key, 0, len(parts))
	for _, part := range parts {
		keyParts = append(keyParts, Key(part))
	}
	return keyParts
}

func (p *PrefixCache[I]) nodeForKey(key Key) *node[I] {
	keyParts := p.splitKey(key)
	if p.rootNode == nil {
		p.rootNode = newNode[I](rootNodeKey)
	}
	currentNode := p.rootNode
	var path []string
	for _, part := range keyParts {
		path = append(path, string(part))
		if child, ok := currentNode.children[part]; ok {
			currentNode = child
		} else {
			child = newNode[I](Key(strings.Join(path, p.separator)))
			currentNode.children[part] = child
			currentNode = child
		}
	}
	return currentNode
}

func (p *PrefixCache[I]) nodesInPath(key Key) []*node[I] {
	if p.rootNode == nil {
		return nil
	}
	currentNode := p.rootNode
	keyParts := p.splitKey(key)
	nodes := make([]*node[I], 0, len(keyParts))
	for _, part := range keyParts {
		if child, ok := currentNode.children[part]; ok {
			nodes = append(nodes, currentNode)
			currentNode = child
		} else {
			// key is not found
			return nil
		}
	}
	nodes = append(nodes, currentNode)
	return nodes
}

type node[I interface{}] struct {
	key      Key
	value    *I
	children map[Key]*node[I]
}

func newNode[I interface{}](keyPart Key) *node[I] {
	return &node[I]{
		key:      keyPart,
		value:    nil,
		children: map[Key]*node[I]{},
	}
}

func newMap[I interface{}]() map[Key]*node[I] {
	return map[Key]*node[I]{}
}
