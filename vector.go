package vector

const (
	bits  = 5 // number of bits needed to represent the range (0 32].
	width = 32
	mask  = width - 1 // 0x1f
)

// Vector is an immutable vector implementation with O(1) lookup,
// insertion, appending, and deletion.
type Vector[T any] struct {
	cnt, shift int
	root, tail *node[T]
}

func New[T any](items ...T) (vec Vector[T]) {
	if len(items) > 0 {
		trans := vec.transient()
		trans.Append(items...)
		vec = trans.Vector()
	}

	return
}

func newVector[T any]() Vector[T] {
	node := &node[T]{}
	return Vector[T]{
		shift: bits,
		root:  node,
		tail:  node,
	}
}

// Transient returns a *Transient with the same value as v.
// The transient vector is mutable, making suitable for optimizing
// tight loops where intermediate values of the transient are not shared.
// When the transient vector has reached the desired state, it should be
// persisted with a call to Persistent() prior to sharing.
func (v Vector[T]) transient() *Builder[T] {
	if v == (Vector[T]{}) {
		v = newVector[T]()
	}

	return &Builder[T]{
		cnt:   v.cnt,
		shift: v.shift,
		root:  v.root.clone(),
		tail:  v.tail.clone(),
	}
}

// Len returns the number of elements contained in the Vector.
func (v Vector[T]) Len() int {
	return v.cnt
}

func (v Vector[T]) tailoff() int {
	if v.cnt < width {
		return 0
	}

	return ((v.cnt - 1) >> bits) << bits
}

func (v Vector[T]) nodeFor(i int) *node[T] {
	if i >= 0 && i < v.cnt {
		if i >= v.tailoff() {
			return v.tail
		}

		n := v.root
		for level := v.shift; level > 0; level -= bits {
			n = n.array[(i>>level)&mask].(*node[T])
		}

		return n
	}

	panic("index out of bounds")
}

// At i returns the ith entry in the Vector
func (v Vector[T]) At(i int) T {
	t, _ := v.nodeFor(i).array[i&mask].(T)
	return t
}

// Set takes a value and "associates" it to the Vector,
// assigning it to the index.
func (v Vector[T]) Set(index int, t T) Vector[T] {
	if index >= 0 && index < v.cnt {
		if index >= v.tailoff() {
			newTail := v.tail.clone()
			newTail.array[index&mask] = t
			return Vector[T]{
				cnt:   v.cnt,
				shift: v.shift,
				root:  v.root,
				tail:  newTail,
			}
		}

		return Vector[T]{
			cnt:   v.cnt,
			shift: v.shift,
			root:  v.doAssoc(v.shift, v.root, index, t),
			tail:  v.tail,
		}
	}

	if index == v.cnt {
		return v.cons(t)
	}

	panic("index out of bounds")
}

func (v Vector[T]) doAssoc(level int, n *node[T], i int, t T) *node[T] {
	ret := n
	if level == 0 {
		ret.array[i&mask] = t
	} else {
		subidx := (i >> level) & mask
		ret.array[subidx] = v.doAssoc(level-bits, n.array[subidx].(*node[T]), i, t)
	}

	return ret
}

// Append values to the Vector.
func (v Vector[T]) Append(ts ...T) Vector[T] {
	switch len(ts) {
	case 0:
		return v

	case 1:
		return v.cons(ts[0])

	default:
		head, ts := ts[0], ts[1:]
		b := v.cons(head).transient()
		b.Append(ts...)
		return b.Vector()
	}
}

func (v Vector[T]) cons(t T) Vector[T] {
	if v == (Vector[T]{}) {
		v = newVector[T]()
	}

	// room in tail?
	if v.cnt-v.tailoff() < 32 {
		newTail := v.tail.clone()
		newTail.len++
		newTail.array[v.tail.len] = t

		return Vector[T]{
			cnt:   v.cnt + 1,
			shift: v.shift,
			root:  v.root,
			tail:  newTail,
		}
	}

	// full tail; push into trie
	newRoot := &node[T]{}
	tailNode := v.tail.clone()
	newShift := v.shift

	// overflow root?
	if (v.cnt >> bits) > (1 << v.shift) {
		newRoot.len += 2
		newRoot.array[0] = v.root
		newRoot.array[1] = newPath(v.shift, tailNode)
		newShift += bits
	} else {
		newRoot = v.pushTail(v.shift, v.root, tailNode)
	}

	return Vector[T]{
		cnt:   v.cnt + 1,
		shift: newShift,
		root:  newRoot,
		tail:  newValueNode(t),
	}
}

func newPath[T any](level int, n *node[T]) *node[T] {
	if level <= 0 {
		return n
	}

	return newPathNode(newPath(level-bits, n))
}

func (v Vector[T]) pushTail(level int, parent, tailNode *node[T]) *node[T] {
	//if parent is leaf, insert node,
	// else does it map to an existing child? -> nodeToInsert = pushNode one more level
	// else alloc new path
	//return  nodeToInsert placed in copy of parent

	subidx := ((v.cnt - 1) >> level) & mask
	ret := parent.clone()

	var nodeToInsert *node[T]

	if level == bits {
		nodeToInsert = tailNode
	} else {
		if child := parent.array[subidx]; child != nil {
			nodeToInsert = v.pushTail(level-bits, child.(*node[T]), tailNode)
		} else {
			nodeToInsert = newPath(level-bits, tailNode)
		}
	}

	ret.array[subidx] = nodeToInsert
	return ret
}

// Pop returns a copy of the Vector without its last element.
func (v Vector[T]) Pop() Vector[T] {
	if v.cnt <= 1 {
		return Vector[T]{}
	}

	// len(tail) > 1 ?
	if v.cnt-v.tailoff() > 1 {
		newTail := &node[T]{len: v.tail.len - 1}
		copy(newTail.array[:newTail.len], v.tail.array[:])

		return Vector[T]{
			cnt:   v.cnt - 1,
			shift: v.shift,
			root:  v.root,
			tail:  newTail,
		}
	}

	newTail := v.nodeFor(v.cnt - 2)

	newRoot := v.popTail(v.shift, v.root)
	newShift := v.shift
	if newRoot == nil {
		newRoot = &node[T]{}
	}
	if v.shift > bits && newRoot.array[1] == nil {
		if newRoot.array[0] == nil {
			newRoot = &node[T]{}
		}
		newShift -= bits
	}

	return Vector[T]{
		cnt:   v.cnt - 1,
		shift: newShift,
		root:  newRoot,
		tail:  newTail,
	}
}

func (v Vector[T]) popTail(level int, n *node[T]) *node[T] {
	subidx := ((v.cnt - 2) >> level) & mask
	if level > bits {
		newChild := v.popTail(level-bits, n.array[subidx].(*node[T]))
		if newChild == nil && subidx == 0 {
			return nil
		}

		ret := n.clone()
		ret.array[subidx] = newChild
		// ret.len++
		return ret

	} else if subidx == 0 {
		return nil
	}

	ret := n.clone()
	ret.array[subidx] = node[T]{}
	return ret
}

// Builder is a mutable Vector that minimizes allocation for all operations.
// Callers MUST NOT share transient objects, nor convert shared Vectors into
// Builders.
type Builder[T any] struct {
	cnt, shift int
	root, tail *node[T]
}

func NewBuilder[T any]() *Builder[T] {
	vec := newVector[T]()
	return (*Builder[T])(&vec)
}

// Vector finalizes the builder into a Vector.
// Users MUST NOT mutate t after a call to Vector.
func (t Builder[T]) Vector() Vector[T] { return (Vector[T])(t) }

func (t Builder[T]) tailoff() int { return (Vector[T])(t).tailoff() }

// Count the number of elements in the vector.
func (t *Builder[T]) Len() int { return t.cnt }

// Append values to the vector
func (t *Builder[T]) Append(ts ...T) {
	for _, val := range ts {
		t.Cons(val)
	}
}

func (t *Builder[T]) Cons(val T) {
	// room in tail?
	if t.cnt-t.tailoff() < 32 {
		t.tail.array[t.cnt&mask] = val
		t.tail.len++
		t.cnt++
		return
	}

	// full tail; push into trie
	newRoot := &node[T]{}
	tailNode := t.tail.clone()
	t.tail = newValueNode(val)
	newShift := t.shift

	// overflow root?
	if (t.cnt >> bits) > (1 << t.shift) {
		newRoot.len += 2
		newRoot.array[0] = t.root
		newRoot.array[1] = newPath(t.shift, tailNode)
		newShift += 5
	} else {
		newRoot = t.pushTail(t.shift, t.root, tailNode)
	}

	t.root = newRoot
	t.shift = newShift
	t.cnt++
}

func (t *Builder[T]) pushTail(level int, parent, tailNode *node[T]) *node[T] {
	//if parent is leaf, insert node,
	// else does it map to an existing child? -> nodeToInsert = pushNode one more level
	// else alloc new path
	//return  nodeToInsert placed in parent

	subidx := ((t.cnt - 1) >> level) & mask
	ret := parent // mutable; don't clone
	var nodeToInsert *node[T]
	if level == bits {
		nodeToInsert = tailNode
	} else {
		if child := parent.array[subidx]; child != nil {
			nodeToInsert = t.pushTail(level-bits, child.(*node[T]), tailNode)
		} else {
			nodeToInsert = newPath(level-bits, tailNode)
		}
	}

	ret.array[subidx] = nodeToInsert
	return ret
}

type node[T any] struct {
	len   int
	array [width]any
}

func newValueNode[T any](vs ...T) *node[T] {
	n := &node[T]{len: len(vs)}
	for i, v := range vs {
		n.array[i] = v
	}

	return n
}

func newPathNode[T any](n *node[T]) *node[T] {
	out := &node[T]{len: 1}
	out.array[0] = n
	return out
}

func (n *node[T]) clone() *node[T] {
	return &node[T]{
		len:   n.len,
		array: n.array,
	}
}
