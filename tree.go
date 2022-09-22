// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// at https://github.com/julienschmidt/httprouter/blob/master/LICENSE

package gin

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin/internal/bytesconv"
)

var (
	strColon = []byte(":")
	strStar  = []byte("*")
	strSlash = []byte("/")
)

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the router.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// Get returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) Get(name string) (string, bool) {
	for _, entry := range ps {
		if entry.Key == name {
			return entry.Value, true
		}
	}
	return "", false
}

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) (va string) {
	va, _ = ps.Get(name)
	return
}

//基数树，radix-tree，由 前缀树 演变 进化而来
type methodTree struct {
	method string
	root   *node
}

//共9个，按照method将所有的方法分开, 然后每个method下面都是一个radix tree
//GET、PUT 、DELETE、 POST 、OPTION、 PATCH、HEAD、TRACE、CONNECT
type methodTrees []methodTree

//tips: 为什么用 数组 不用 map？
func (trees methodTrees) get(method string) *node {
	for _, tree := range trees {
		if tree.method == method {
			return tree.root
		}
	}
	return nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

//最长 公共子前缀
func longestCommonPrefix(a, b string) int {
	i := 0
	max := min(len(a), len(b))
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}

// addChild will add a child node, keeping wildcards at the end
func (n *node) addChild(child *node) {
	if IsDebugging() {
		fmt.Println("\n ==> 插入到子节点【addChild】 node== \n", child)
	}
	if n.wildChild && len(n.children) > 0 {
		wildcardChild := n.children[len(n.children)-1]
		n.children = append(n.children[:len(n.children)-1], child, wildcardChild)
	} else {
		n.children = append(n.children, child)
	}
}

func countParams(path string) uint16 {
	var n uint16
	s := bytesconv.StringToBytes(path)
	n += uint16(bytes.Count(s, strColon))
	n += uint16(bytes.Count(s, strStar))
	return n
}

func countSections(path string) uint16 {
	s := bytesconv.StringToBytes(path)
	return uint16(bytes.Count(s, strSlash))
}

type nodeType uint8

const (
	static   nodeType = iota // default
	root                     //根节点
	param                    //参数节点
	catchAll                 //通配符，节点，必须在路径的最后，
	/*
		catchAll 举例：
		比如 /srcfilepath
		/src/                     match
		/src/somefile.go          match
		/src/subdir/somefile.go   match
	*/
)

func (n nodeType) String() (res string) {
	switch uint8(n) {
	case 0:
		res = "0=nodeType=默认类型"
	case 1:
		res = " 🌲 🌲 1=root=根节点 🌲 🌲"
	case 2:
		res = "2=param=参数节点"
	case 3:
		res = "3=catchAll=通配符节点"
	}
	return
}

/*----------------------------------
//----------GET请求树例子-------------
//----------------------------------
Priority   Path             Handle
9          \                *<1>
3          ├s               nil
2          |├earch\         *<2>
1          |└upport\        *<3>
2          ├blog\           *<4>
1          |    └:post      nil
1          |         └\     *<5>
2          ├about-us\       *<6>
1          |        └team\  *<7>
1          └contact\        *<8>
*/
//radix-tree 节点 类型，类似的上次说 apisex也是用的这个数据结构，所以apisex说 时间复杂度是O(K),与K自身长度有关，http源码用的是map哈希， radix-tree基数树 内存小， 也叫 压缩字典树
type node struct {
	//这个节点的URL的路径
	//例如search与support，共同的父节点path='s'，类型就是static
	//子节点就是2个，'earch'和'upport'
	path string

	//保存所有子节点的第一个字符，例如search与support，indices保存的是eu，标识有2个 子节点，子节点分支分别是e、u
	indices   string
	wildChild bool //是否参数 节点

	nType    nodeType
	priority uint32  //权重，优先级，便于查找
	children []*node // child nodes, at most 1 :param style node at the end of the array
	handlers HandlersChain
	fullPath string
}

//每一层的节点按照priority排序，一个节点的priority值表示他包含的所有子节点（子节点，孙节点等）的数量，这样做有两个好处：
//1. 被最多路径包含的节点会被最先评估。这样可以让尽量多的路由快速被定位。
//2. 有点像成本补偿。最长的路径可以被最先评估，补偿体现在最长的路径需要花费更长的时间来定位，如果最长路径的节点能被优先评估（即每次拿子节点都命中），那么所花时间不一定比短路径的路由长。

//自定义打印
func (h HandlersChain) String() string {
	return namesOfFunctions(h)
}

//自定义打印
func (n *node) String() string {
	//fmt.Println(" 这是路有树: \n  " + n.FormatTree())
	return fmt.Sprintf("⭐️⭐️⭐️[打印当前node]=⭐️⭐️⭐️   \n %s, %+#v \n 节点类型:%s,  \n 其中handlers有 [%-7s] \n ", n.FormatTree(), n, n.nType, n.handlers)
}

// FormatTree 格式化树结构
// --
//   |__ a
//     |__ bd
//     |__ d
func (n *node) FormatTree() string {
	var buf bytes.Buffer
	if n.path == "" {
		buf.WriteString("\n-- \n")
	} else {
		buf.WriteString("|__ " + n.path + "\n")
	}
	if n.children != nil {
		for _, child := range n.children {
			childStr := child.FormatTree()
			// 增加前缀
			// 分割行
			sps := strings.Split(childStr, "\n")
			for _, sp := range sps {
				if sp != "" {
					buf.WriteString("  ")
					buf.WriteString(sp)
					buf.WriteString("\n")
				}
			}
		}
	}
	return buf.String()
}

// 增加指定孩子节点的优先级，并更新节点的indices
// 这并不会影响路由功能，但是可以加快孩子节点的查找速度
// Increments priority of the given child and reorders if necessary
func (n *node) incrementChildPrio(pos int) int {
	cs := n.children
	cs[pos].priority++
	prio := cs[pos].priority

	// Adjust position (move to front)	// 将更新后的priority向前移动，保持按优先级降序排列
	newPos := pos
	for ; newPos > 0 && cs[newPos-1].priority < prio; newPos-- {
		// Swap node positions
		cs[newPos-1], cs[newPos] = cs[newPos], cs[newPos-1]
	}

	// Build new index char string	// 根据优先级重新构建indices，indices保存着当前节点下的每个孩子节点的首字符
	if newPos != pos {
		n.indices = n.indices[:newPos] + // Unchanged prefix, might be empty
			n.indices[pos:pos+1] + // The index char we move
			n.indices[newPos:pos] + n.indices[pos+1:] // Rest without char at 'pos'
	}

	return newPos
}

//建树
// addRoute adds a node with the given handle to the path.
// Not concurrency-safe!
//添加路由的逻辑有点绕，简而言之就是  找到正确的位置  调用insertChild 将新的节点加到树中
func (n *node) addRoute(path string, handlers HandlersChain) {
	fullPath := path
	n.priority++

	// Empty tree
	if len(n.path) == 0 && len(n.children) == 0 {
		n.insertChild(path, fullPath, handlers)
		n.nType = root //根节点
		return
	}

	parentFullPathIndex := 0

walk:
	for {
		// Find the longest common prefix.
		// This also implies that the common prefix contains no ':' or '*'
		// since the existing key can't contain those chars.
		i := longestCommonPrefix(path, n.path) //找到最长公共子前缀，然后 进行分裂：

		// Split edge
		//比如 /search与/support，最长公共子前缀是/s，则/s是父节点，非公共部分为子节点是eu, /s为新节点，eu 保存原来节点信息
		if i < len(n.path) {
			child := node{
				path:      n.path[i:], //非公共的部分
				wildChild: n.wildChild,
				indices:   n.indices,
				children:  n.children,
				handlers:  n.handlers,
				priority:  n.priority - 1,
				fullPath:  n.fullPath,
			}

			n.children = []*node{&child}
			// []byte for proper unicode char conversion, see #65
			n.indices = bytesconv.BytesToString([]byte{n.path[i]})
			n.path = path[:i]
			n.handlers = nil
			n.wildChild = false
			n.fullPath = fullPath[:parentFullPathIndex+i]
		}

		// Make new node a child of this node
		// 比如 /search与/support，则进入if语句
		// 在当前节点创建一个新的子节点
		if i < len(path) {
			path = path[i:]
			c := path[0]

			// '/' after param
			if n.nType == param && c == '/' && len(n.children) == 1 {
				parentFullPathIndex += len(n.path)
				n = n.children[0] //当前节点等于子节点
				n.priority++      // 对应子节点优先级加1
				fmt.Printf("[当前addRoute][n.nType == param && c == '/' ] node==%#+v \n", n)
				continue walk
			}

			// Check if a child with the next path byte exists	// 循环查找，n.indices记录着所有孩子节点的第一个字符
			for i, max := 0, len(n.indices); i < max; i++ {
				if c == n.indices[i] { //如果找到和当前要插入节点的第一个字符相符，匹配成功
					parentFullPathIndex += len(n.path)
					i = n.incrementChildPrio(i) // 对应子节点优先级加1，并对该子节点的indices重新排列
					n = n.children[i]
					fmt.Printf("[当前addRoute][ i, max := 0, len(n.indices);] node==%#+v \n", n)
					continue walk
				}
			}

			// Otherwise insert it    // 如果添加的节点既不是 * 也不是:这样的通配节点,，，，插入
			if c != ':' && c != '*' && n.nType != catchAll { //默认static节点
				// []byte for proper unicode char conversion, see #65
				n.indices += bytesconv.BytesToString([]byte{c})
				child := &node{
					fullPath: fullPath,
				}
				n.addChild(child)
				n.incrementChildPrio(len(n.indices) - 1)
				n = child
				fmt.Printf("[当前addRoute][n.nType == 默认static节点 ] node==%#+v \n", n)
			} else if n.wildChild { //参数节点, :或者*
				// inserting a wildcard node, need to check if it conflicts with the existing wildcard
				n = n.children[len(n.children)-1]
				n.priority++

				// Check if the wildcard matches
				// 此时的path 已经取成了公共前缀 后的部分
				// 例如原来的路径是/usr/:name，假设当前n节点的父节点为n father
				// 而n在前面已经取成了n father孩子节点
				// 目前情况是nfather.path=/usr，由于其子节点是通配符节点
				// 故nfather.wildChild=true，n.path=/:name
				// 假设新加进来的节点path=/:nameserver
				//则符合这里的if条件，跳转到walk，以n为父节点继续匹配
				if len(path) >= len(n.path) && n.path == path[:len(n.path)] &&
					// Adding a child to a catchAll is not possible// 不可能在全匹配节点（例如*name）后继续加子节点
					n.nType != catchAll &&
					// Check for longer wildcard, e.g. :name and :names
					//检查 是不是 更长的参数，比如 /einstein-logic/v1/user/:userId,与/einstein-logic/v1/user/:userIdssss
					(len(n.path) >= len(path) || path[len(n.path)] == '/') {
					continue walk
				}

				// Wildcard conflict
				pathSeg := path
				if n.nType != catchAll {
					pathSeg = strings.SplitN(pathSeg, "/", 2)[0]
				}
				prefix := fullPath[:strings.Index(fullPath, pathSeg)] + n.path
				panic("'" + pathSeg +
					"' in new path '" + fullPath +
					"' conflicts with existing wildcard '" + n.path +
					"' in existing prefix '" + prefix +
					"'")
			}

			//插入节点
			n.insertChild(path, fullPath, handlers)
			return
		}

		// Otherwise add handle to current node
		//相同路径，直接替换handlers
		if n.handlers != nil {
			panic("handlers are already registered for path '" + fullPath + "'")
		}
		n.handlers = handlers
		n.fullPath = fullPath
		return
	}
}

// Search for a wildcard segment and check the name for invalid characters.
// Returns -1 as index, if no wildcard was found.
// wildcard-通配符字符串（例如:name,wildcard就为name） i-通配符在path的索引 valid-是否有合法的通配符
func findWildcard(path string) (wildcard string, i int, valid bool) {
	// Find start
	for start, c := range []byte(path) {
		// A wildcard starts with ':' (param)参数  or '*' (catch-all) 通配符
		if c != ':' && c != '*' {
			continue
		}

		// Find end and check for invalid characters
		valid = true
		// ":" 或"*"必须先有"/", 不能直接有 ":","*"
		for end, c := range []byte(path[start+1:]) {
			switch c {
			case '/':
				return path[start : start+1+end], start, valid
			case ':', '*': //一个通配符后还有一个通配符，valid置为false
				valid = false
			}
		}
		return path[start:], start, valid
	}
	return "", -1, false
}

//插入子节点
func (n *node) insertChild(path string, fullPath string, handlers HandlersChain) {
	for {
		// Find prefix until first wildcard
		wildcard, i, valid := findWildcard(path)
		//终止条件： 不再有 通配符
		if i < 0 { // No wildcard found
			if IsDebugging() {
				fmt.Println("[创建 路由树]【insertChild终止】", path, "==handlers==", handlers)
			}
			break
		}
		if IsDebugging() {
			fmt.Printf("[创建 路由树][insertChild] path=%v, fullPath=%v, handlers=%#v, \n", path, fullPath, handlers)
		}
		// The wildcard name must not contain ':' and '*'
		if !valid {
			panic("only one wildcard per path segment is allowed, has: '" +
				wildcard + "' in path '" + fullPath + "'")
		}

		// check if the wildcard has a name
		if len(wildcard) < 2 {
			panic("wildcards must be named with a non-empty name in path '" + fullPath + "'")
		}

		if wildcard[0] == ':' { // param
			if i > 0 {
				// Insert prefix before the current wildcard
				n.path = path[:i]
				path = path[i:]
			}
			//参数类型 ":"
			child := &node{
				nType:    param,
				path:     wildcard,
				fullPath: fullPath,
			}
			n.addChild(child)
			n.wildChild = true
			n = child
			n.priority++

			// if the path doesn't end with the wildcard, then there
			// will be another non-wildcard subpath starting with '/'
			// 如果通配符后面还有字符，则一定以/为开头
			// 例如/:name/age 通配符后还有/age
			// 这里的英文注释说“将会有另一个以'/'开头的非通配符子路径”
			// 这不代表不能处理/:name/*hobby这种，上面已经展示了会将通配符的前面部分
			// 设为父节点，也就是说通配符节点的父节点一定是一个非通配符节点，英文的注释应该这么理解的
			if len(wildcard) < len(path) {
				path = path[len(wildcard):]

				child := &node{
					priority: 1,
					fullPath: fullPath,
				}
				n.addChild(child)
				n = child
				continue
			}

			// Otherwise we're done. Insert the handle in the new leaf
			n.handlers = handlers
			return
		}

		// catchAll		// 通配符不是:那么就是*，因为*是全匹配的通配符，那么这种情况是不允许的/*name/pwd，*必须在最后
		if i+len(wildcard) != len(path) {
			panic("catch-all routes are only allowed at the end of the path in path '" + fullPath + "'")
		}

		if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
			panic("catch-all conflicts with existing handle for the path segment root in path '" + fullPath + "'")
		}

		// currently fixed width 1 for '/'
		i--
		if path[i] != '/' {
			panic("no / before catch-all in path '" + fullPath + "'")
		}

		n.path = path[:i]

		// First node: catchAll node with empty path
		//通配符类型 *		// *可以匹配0个或多个字符，第一个节点保存为空，也就是*匹配0个字符的情况
		child := &node{
			wildChild: true,
			nType:     catchAll,
			fullPath:  fullPath,
		}

		n.addChild(child)
		n.indices = string('/')
		n = child
		n.priority++

		// second node: node holding the variable
		// 匹配多个字符的情况
		child = &node{
			path:     path[i:],
			nType:    catchAll,
			handlers: handlers,
			priority: 1,
			fullPath: fullPath,
		}
		n.children = []*node{child}

		return
	}

	// If no wildcard was found, simply insert the path and handle
	n.path = path
	n.handlers = handlers
	n.fullPath = fullPath
}

// nodeValue holds return values of (*Node).getValue method
type nodeValue struct {
	handlers HandlersChain
	params   *Params
	tsr      bool
	fullPath string
}

type skippedNode struct {
	path        string
	node        *node
	paramsCount int16
}

// Returns the handle registered with the given path (key). The values of
// wildcards are saved to a map.
// If no handle can be found, a TSR (trailing slash redirect) recommendation is
// made if a handle exists with an extra (without the) trailing slash for the
// given path.
func (n *node) getValue(path string, params *Params, skippedNodes *[]skippedNode, unescape bool) (value nodeValue) {
	var globalParamsCount int16
	if IsDebugging() {
		fmt.Printf("[getValue] path=%v ,params=%+v,skippedNodes=%+v \n", path, params, skippedNodes)
	}
walk: // Outer loop for walking the tree
	for {
		prefix := n.path
		if len(path) > len(prefix) {
			if path[:len(prefix)] == prefix {
				path = path[len(prefix):]

				// Try all the non-wildcard children first by matching the indices
				idxc := path[0]
				for i, c := range []byte(n.indices) {
					if c == idxc {
						//  strings.HasPrefix(n.children[len(n.children)-1].path, ":") == n.wildChild
						if n.wildChild {
							index := len(*skippedNodes)
							*skippedNodes = (*skippedNodes)[:index+1]
							(*skippedNodes)[index] = skippedNode{
								path: prefix + path,
								node: &node{
									path:      n.path,
									wildChild: n.wildChild,
									nType:     n.nType,
									priority:  n.priority,
									children:  n.children,
									handlers:  n.handlers,
									fullPath:  n.fullPath,
								},
								paramsCount: globalParamsCount,
							}
						}

						n = n.children[i]
						continue walk
					}
				}

				//不是参数节点
				if !n.wildChild {
					// If the path at the end of the loop is not equal to '/' and the current node has no child nodes
					// the current node needs to roll back to last vaild skippedNode
					if path != "/" {
						for l := len(*skippedNodes); l > 0; {
							skippedNode := (*skippedNodes)[l-1]
							*skippedNodes = (*skippedNodes)[:l-1]
							if strings.HasSuffix(skippedNode.path, path) {
								path = skippedNode.path
								n = skippedNode.node
								if value.params != nil {
									*value.params = (*value.params)[:skippedNode.paramsCount]
								}
								globalParamsCount = skippedNode.paramsCount
								continue walk
							}
						}
					}

					// Nothing found.
					// We can recommend to redirect to the same URL without a
					// trailing slash if a leaf exists for that path.
					value.tsr = path == "/" && n.handlers != nil
					return
				}

				// Handle wildcard child, which is always at the end of the array
				n = n.children[len(n.children)-1]
				globalParamsCount++

				switch n.nType {
				case param: //参数类型
					// fix truncate the parameter
					// tree_test.go  line: 204

					// Find param end (either '/' or path end)
					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					// Save param value
					if params != nil && cap(*params) > 0 {
						if value.params == nil {
							value.params = params
						}
						// Expand slice within preallocated capacity
						i := len(*value.params)
						*value.params = (*value.params)[:i+1]
						val := path[:end]
						if unescape {
							if v, err := url.QueryUnescape(val); err == nil {
								val = v
							}
						}
						(*value.params)[i] = Param{
							Key:   n.path[1:],
							Value: val,
						}
					}

					// we need to go deeper!
					if end < len(path) {
						if len(n.children) > 0 {
							path = path[end:]
							n = n.children[0]
							continue walk
						}

						// ... but we can't
						value.tsr = len(path) == end+1
						return
					}

					if value.handlers = n.handlers; value.handlers != nil {
						value.fullPath = n.fullPath
						return
					}
					if len(n.children) == 1 {
						// No handle found. Check if a handle for this path + a
						// trailing slash exists for TSR recommendation
						n = n.children[0]
						value.tsr = n.path == "/" && n.handlers != nil
					}
					return

				case catchAll:
					// Save param value
					if params != nil {
						if value.params == nil {
							value.params = params
						}
						// Expand slice within preallocated capacity
						i := len(*value.params)
						*value.params = (*value.params)[:i+1]
						val := path
						if unescape {
							if v, err := url.QueryUnescape(path); err == nil {
								val = v
							}
						}
						(*value.params)[i] = Param{
							Key:   n.path[2:],
							Value: val,
						}
					}

					value.handlers = n.handlers
					value.fullPath = n.fullPath
					return

				default:
					panic("invalid node type")
				}
			}
		}

		if path == prefix {
			// If the current path does not equal '/' and the node does not have a registered handle and the most recently matched node has a child node
			// the current node needs to roll back to last vaild skippedNode
			if n.handlers == nil && path != "/" {
				for l := len(*skippedNodes); l > 0; {
					skippedNode := (*skippedNodes)[l-1]
					*skippedNodes = (*skippedNodes)[:l-1]
					if strings.HasSuffix(skippedNode.path, path) {
						path = skippedNode.path
						n = skippedNode.node
						if value.params != nil {
							*value.params = (*value.params)[:skippedNode.paramsCount]
						}
						globalParamsCount = skippedNode.paramsCount
						continue walk
					}
				}
				//	n = latestNode.children[len(latestNode.children)-1]
			}
			// We should have reached the node containing the handle.
			// Check if this node has a handle registered.
			if value.handlers = n.handlers; value.handlers != nil {
				value.fullPath = n.fullPath
				return
			}

			// If there is no handle for this route, but this route has a
			// wildcard child, there must be a handle for this path with an
			// additional trailing slash
			if path == "/" && n.wildChild && n.nType != root {
				value.tsr = true
				return
			}

			// No handle found. Check if a handle for this path + a
			// trailing slash exists for trailing slash recommendation
			for i, c := range []byte(n.indices) {
				if c == '/' {
					n = n.children[i]
					value.tsr = (len(n.path) == 1 && n.handlers != nil) ||
						(n.nType == catchAll && n.children[0].handlers != nil)
					return
				}
			}

			return
		}

		// Nothing found. We can recommend to redirect to the same URL with an
		// extra trailing slash if a leaf exists for that path
		value.tsr = path == "/" ||
			(len(prefix) == len(path)+1 && prefix[len(path)] == '/' &&
				path == prefix[:len(prefix)-1] && n.handlers != nil)

		// roll back to last valid skippedNode
		if !value.tsr && path != "/" {
			for l := len(*skippedNodes); l > 0; {
				skippedNode := (*skippedNodes)[l-1]
				*skippedNodes = (*skippedNodes)[:l-1]
				if strings.HasSuffix(skippedNode.path, path) {
					path = skippedNode.path
					n = skippedNode.node
					if value.params != nil {
						*value.params = (*value.params)[:skippedNode.paramsCount]
					}
					globalParamsCount = skippedNode.paramsCount
					continue walk
				}
			}
		}

		return
	}
}

// Makes a case-insensitive lookup of the given path and tries to find a handler.
// It can optionally also fix trailing slashes.
// It returns the case-corrected path and a bool indicating whether the lookup
// was successful.
func (n *node) findCaseInsensitivePath(path string, fixTrailingSlash bool) ([]byte, bool) {
	const stackBufSize = 128

	// Use a static sized buffer on the stack in the common case.
	// If the path is too long, allocate a buffer on the heap instead.
	buf := make([]byte, 0, stackBufSize)
	if length := len(path) + 1; length > stackBufSize {
		buf = make([]byte, 0, length)
	}

	ciPath := n.findCaseInsensitivePathRec(
		path,
		buf,       // Preallocate enough memory for new path
		[4]byte{}, // Empty rune buffer
		fixTrailingSlash,
	)

	return ciPath, ciPath != nil
}

// Shift bytes in array by n bytes left
func shiftNRuneBytes(rb [4]byte, n int) [4]byte {
	switch n {
	case 0:
		return rb
	case 1:
		return [4]byte{rb[1], rb[2], rb[3], 0}
	case 2:
		return [4]byte{rb[2], rb[3]}
	case 3:
		return [4]byte{rb[3]}
	default:
		return [4]byte{}
	}
}

// Recursive case-insensitive lookup function used by n.findCaseInsensitivePath
func (n *node) findCaseInsensitivePathRec(path string, ciPath []byte, rb [4]byte, fixTrailingSlash bool) []byte {
	npLen := len(n.path)

walk: // Outer loop for walking the tree
	for len(path) >= npLen && (npLen == 0 || strings.EqualFold(path[1:npLen], n.path[1:])) {
		// Add common prefix to result
		oldPath := path
		path = path[npLen:]
		ciPath = append(ciPath, n.path...)

		if len(path) == 0 {
			// We should have reached the node containing the handle.
			// Check if this node has a handle registered.
			if n.handlers != nil {
				return ciPath
			}

			// No handle found.
			// Try to fix the path by adding a trailing slash
			if fixTrailingSlash {
				for i, c := range []byte(n.indices) {
					if c == '/' {
						n = n.children[i]
						if (len(n.path) == 1 && n.handlers != nil) ||
							(n.nType == catchAll && n.children[0].handlers != nil) {
							return append(ciPath, '/')
						}
						return nil
					}
				}
			}
			return nil
		}

		// If this node does not have a wildcard (param or catchAll) child,
		// we can just look up the next child node and continue to walk down
		// the tree
		if !n.wildChild {
			// Skip rune bytes already processed
			rb = shiftNRuneBytes(rb, npLen)

			if rb[0] != 0 {
				// Old rune not finished
				idxc := rb[0]
				for i, c := range []byte(n.indices) {
					if c == idxc {
						// continue with child node
						n = n.children[i]
						npLen = len(n.path)
						continue walk
					}
				}
			} else {
				// Process a new rune
				var rv rune

				// Find rune start.
				// Runes are up to 4 byte long,
				// -4 would definitely be another rune.
				var off int
				for max := min(npLen, 3); off < max; off++ {
					if i := npLen - off; utf8.RuneStart(oldPath[i]) {
						// read rune from cached path
						rv, _ = utf8.DecodeRuneInString(oldPath[i:])
						break
					}
				}

				// Calculate lowercase bytes of current rune
				lo := unicode.ToLower(rv)
				utf8.EncodeRune(rb[:], lo)

				// Skip already processed bytes
				rb = shiftNRuneBytes(rb, off)

				idxc := rb[0]
				for i, c := range []byte(n.indices) {
					// Lowercase matches
					if c == idxc {
						// must use a recursive approach since both the
						// uppercase byte and the lowercase byte might exist
						// as an index
						if out := n.children[i].findCaseInsensitivePathRec(
							path, ciPath, rb, fixTrailingSlash,
						); out != nil {
							return out
						}
						break
					}
				}

				// If we found no match, the same for the uppercase rune,
				// if it differs
				if up := unicode.ToUpper(rv); up != lo {
					utf8.EncodeRune(rb[:], up)
					rb = shiftNRuneBytes(rb, off)

					idxc := rb[0]
					for i, c := range []byte(n.indices) {
						// Uppercase matches
						if c == idxc {
							// Continue with child node
							n = n.children[i]
							npLen = len(n.path)
							continue walk
						}
					}
				}
			}

			// Nothing found. We can recommend to redirect to the same URL
			// without a trailing slash if a leaf exists for that path
			if fixTrailingSlash && path == "/" && n.handlers != nil {
				return ciPath
			}
			return nil
		}

		n = n.children[0]
		switch n.nType {
		case param:
			// Find param end (either '/' or path end)
			end := 0
			for end < len(path) && path[end] != '/' {
				end++
			}

			// Add param value to case insensitive path
			ciPath = append(ciPath, path[:end]...)

			// We need to go deeper!
			if end < len(path) {
				if len(n.children) > 0 {
					// Continue with child node
					n = n.children[0]
					npLen = len(n.path)
					path = path[end:]
					continue
				}

				// ... but we can't
				if fixTrailingSlash && len(path) == end+1 {
					return ciPath
				}
				return nil
			}

			if n.handlers != nil {
				return ciPath
			}

			if fixTrailingSlash && len(n.children) == 1 {
				// No handle found. Check if a handle for this path + a
				// trailing slash exists
				n = n.children[0]
				if n.path == "/" && n.handlers != nil {
					return append(ciPath, '/')
				}
			}

			return nil

		case catchAll:
			return append(ciPath, path...)

		default:
			panic("invalid node type")
		}
	}

	// Nothing found.
	// Try to fix the path by adding / removing a trailing slash
	if fixTrailingSlash {
		if path == "/" {
			return ciPath
		}
		if len(path)+1 == npLen && n.path[len(path)] == '/' &&
			strings.EqualFold(path[1:], n.path[1:len(path)]) && n.handlers != nil {
			return append(ciPath, n.path...)
		}
	}
	return nil
}

func (n *node) Search1() {
	if !IsDebugging() {
		return
	}
	if n == nil {
		return
	}
	fmt.Printf("[node.Search1][打印当前 node 节点 信息] %+#v  \n", n)
	for i, _ := range n.children {
		n.children[i].Search1()
	}
}
