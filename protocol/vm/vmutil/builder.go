package vmutil

import (
	"encoding/binary"

	"chain/errors"
	"chain/protocol/vm"
)

type Builder struct {
	program     []byte
	jumpCounter int

	// Maps a jump target number to its absolute address.
	jumpAddr map[int]uint32

	// Maps a jump target number to the list of places where its
	// absolute address must be filled in once known.
	jumpPlaceholders map[int][]int
}

func NewBuilder() *Builder {
	return &Builder{
		jumpAddr:         make(map[int]uint32),
		jumpPlaceholders: make(map[int][]int),
	}
}

// AddInt64 adds a pushdata instruction for an integer value.
func (b *Builder) AddInt64(n int64) *Builder {
	b.program = append(b.program, vm.PushdataInt64(n)...)
	return b
}

// AddData adds a pushdata instruction for a given byte string.
func (b *Builder) AddData(data []byte) *Builder {
	b.program = append(b.program, vm.PushdataBytes(data)...)
	return b
}

// AddRawBytes simply appends the given bytes to the program. (It does
// not introduce a pushdata opcode.)
func (b *Builder) AddRawBytes(data []byte) *Builder {
	b.program = append(b.program, data...)
	return b
}

// AddOp adds the given opcode to the program.
func (b *Builder) AddOp(op vm.Op) *Builder {
	b.program = append(b.program, byte(op))
	return b
}

// NewJumpTarget allocates a number that can be used as a jump target
// in AddJump and AddJumpIf. Call SetJumpTarget to associate the
// number with a program location.
func (b *Builder) NewJumpTarget() int {
	b.jumpCounter++
	return b.jumpCounter
}

// AddJump adds a JUMP opcode whose target is the given target
// number. The actual program location of the target does not need to
// be known yet, as long as SetJumpTarget is called before Build.
func (b *Builder) AddJump(target int) *Builder {
	return b.addJump(vm.OP_JUMP, target)
}

// AddJumpIf adds a JUMPIF opcode whose target is the given target
// number. The actual program location of the target does not need to
// be known yet, as long as SetJumpTarget is called before Build.
func (b *Builder) AddJumpIf(target int) *Builder {
	return b.addJump(vm.OP_JUMPIF, target)
}

func (b *Builder) addJump(op vm.Op, target int) *Builder {
	b.AddOp(op)
	b.jumpPlaceholders[target] = append(b.jumpPlaceholders[target], len(b.program))
	b.AddRawBytes([]byte{0, 0, 0, 0})
	return b
}

// SetJumpTarget associates the given jump-target number with the
// current position in the program - namely, the program's length,
// such that the first instruction executed by a jump using this
// target will be whatever instruction is added next. It is legal for
// SetJumpTarget to be called at the end of the program, causing jumps
// using that target to fall off the end. There must be a call to
// SetJumpTarget for every jump target used before any call to Build.
func (b *Builder) SetJumpTarget(target int) *Builder {
	b.jumpAddr[target] = uint32(len(b.program))
	return b
}

var ErrUnresolvedJump = errors.New("unresolved jump target")

// Build produces the bytecode of the program. It first resolves any
// jumps in the program by filling in the addresses of their
// targets. This requires SetJumpTarget to be called prior to Build
// for each jump target used (in a call to AddJump or AddJumpIf). If
// any target's address hasn't been set in this way, this function
// produces ErrUnresolvedJump. There are no other error conditions.
func (b *Builder) Build() ([]byte, error) {
	for target, placeholders := range b.jumpPlaceholders {
		addr, ok := b.jumpAddr[target]
		if !ok {
			return nil, errors.Wrapf(ErrUnresolvedJump, "target %d", target)
		}
		for _, placeholder := range placeholders {
			binary.LittleEndian.PutUint32(b.program[placeholder:placeholder+4], addr)
		}
	}
	return b.program, nil
}
