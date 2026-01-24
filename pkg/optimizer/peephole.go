package optimizer

import (
	"github.com/ytnobody/calcium-lang/pkg/bytecode"
)

// PeepholeOptimize applies peephole optimizations to bytecode instructions.
// It looks for small patterns of instructions that can be replaced with
// more efficient sequences.
func PeepholeOptimize(instructions bytecode.Instructions) bytecode.Instructions {
	if len(instructions) == 0 {
		return instructions
	}

	// Parse instructions into a structured format for easier manipulation
	parsed := parseInstructions(instructions)

	// Apply optimization passes
	optimized := parsed
	for {
		newOptimized := applyPeepholePass(optimized)
		if len(newOptimized) == len(optimized) {
			// No more optimizations possible
			break
		}
		optimized = newOptimized
	}

	// Fix jump offsets after optimization
	optimized = fixJumpOffsets(optimized)

	// Convert back to bytecode
	return serializeInstructions(optimized)
}

// instruction represents a parsed bytecode instruction
type instruction struct {
	opcode   bytecode.OpCode
	operands []int
	offset   int // Original byte offset (for jump target resolution)
}

// parseInstructions converts bytecode into a slice of instructions
func parseInstructions(ins bytecode.Instructions) []instruction {
	var result []instruction
	i := 0
	for i < len(ins) {
		op := bytecode.OpCode(ins[i])
		def, err := bytecode.Lookup(op)
		if err != nil {
			// Unknown opcode, skip it
			i++
			continue
		}

		operands, read := bytecode.ReadOperands(def, ins[i+1:])
		result = append(result, instruction{
			opcode:   op,
			operands: operands,
			offset:   i,
		})
		i += 1 + read
	}
	return result
}

// serializeInstructions converts instructions back to bytecode
func serializeInstructions(instructions []instruction) bytecode.Instructions {
	var result bytecode.Instructions
	for _, inst := range instructions {
		result = append(result, bytecode.Make(inst.opcode, inst.operands...)...)
	}
	return result
}

// applyPeepholePass applies one round of peephole optimizations
func applyPeepholePass(instructions []instruction) []instruction {
	if len(instructions) == 0 {
		return instructions
	}

	result := make([]instruction, 0, len(instructions))
	i := 0

	for i < len(instructions) {
		// Try to match patterns (longer patterns first)
		if matched, replacement := matchPattern(instructions, i); matched > 0 {
			result = append(result, replacement...)
			i += matched
			continue
		}

		// No pattern matched, keep the instruction
		result = append(result, instructions[i])
		i++
	}

	return result
}

// matchPattern tries to match optimization patterns at the given position
// Returns (number of instructions matched, replacement instructions)
func matchPattern(instructions []instruction, pos int) (int, []instruction) {
	remaining := len(instructions) - pos

	// Pattern: OpConstant + OpPop -> (remove both if no side effects)
	// This removes unused constants
	if remaining >= 2 {
		if instructions[pos].opcode == bytecode.OpConstant &&
			instructions[pos+1].opcode == bytecode.OpPop {
			// Remove both instructions (dead constant)
			return 2, nil
		}
	}

	// Pattern: OpDup + OpPop -> (remove both)
	if remaining >= 2 {
		if instructions[pos].opcode == bytecode.OpDup &&
			instructions[pos+1].opcode == bytecode.OpPop {
			return 2, nil
		}
	}

	// Pattern: OpTrue + OpNot -> OpFalse
	if remaining >= 2 {
		if instructions[pos].opcode == bytecode.OpTrue &&
			instructions[pos+1].opcode == bytecode.OpNot {
			return 2, []instruction{{opcode: bytecode.OpFalse}}
		}
	}

	// Pattern: OpFalse + OpNot -> OpTrue
	if remaining >= 2 {
		if instructions[pos].opcode == bytecode.OpFalse &&
			instructions[pos+1].opcode == bytecode.OpNot {
			return 2, []instruction{{opcode: bytecode.OpTrue}}
		}
	}

	// Pattern: OpNot + OpNot -> (remove both, double negation)
	if remaining >= 2 {
		if instructions[pos].opcode == bytecode.OpNot &&
			instructions[pos+1].opcode == bytecode.OpNot {
			return 2, nil
		}
	}

	// Pattern: OpNeg + OpNeg -> (remove both, double negation)
	if remaining >= 2 {
		if instructions[pos].opcode == bytecode.OpNeg &&
			instructions[pos+1].opcode == bytecode.OpNeg {
			return 2, nil
		}
	}

	// Pattern: OpJump to next instruction -> (remove)
	if remaining >= 1 {
		if instructions[pos].opcode == bytecode.OpJump {
			jumpTarget := instructions[pos].operands[0]
			// Calculate what the next instruction's offset would be
			def, _ := bytecode.Lookup(bytecode.OpJump)
			nextOffset := instructions[pos].offset + 1
			for _, w := range def.OperandWidths {
				nextOffset += w
			}
			if jumpTarget == nextOffset {
				// Jump to the very next instruction, remove it
				return 1, nil
			}
		}
	}

	// Pattern: OpSetLocal X + OpGetLocal X -> OpDup + OpSetLocal X
	// (when we immediately read what we just wrote)
	if remaining >= 2 {
		if instructions[pos].opcode == bytecode.OpSetLocal &&
			instructions[pos+1].opcode == bytecode.OpGetLocal &&
			instructions[pos].operands[0] == instructions[pos+1].operands[0] {
			return 2, []instruction{
				{opcode: bytecode.OpDup},
				{opcode: bytecode.OpSetLocal, operands: instructions[pos].operands},
			}
		}
	}

	// Pattern: OpSetGlobal X + OpGetGlobal X -> OpDup + OpSetGlobal X
	if remaining >= 2 {
		if instructions[pos].opcode == bytecode.OpSetGlobal &&
			instructions[pos+1].opcode == bytecode.OpGetGlobal &&
			instructions[pos].operands[0] == instructions[pos+1].operands[0] {
			return 2, []instruction{
				{opcode: bytecode.OpDup},
				{opcode: bytecode.OpSetGlobal, operands: instructions[pos].operands},
			}
		}
	}

	return 0, nil
}

// fixJumpOffsets recalculates jump offsets after optimization
func fixJumpOffsets(instructions []instruction) []instruction {
	if len(instructions) == 0 {
		return instructions
	}

	// Build a map from old offsets to new offsets
	oldToNew := make(map[int]int)
	currentOffset := 0

	// First pass: calculate new offsets
	for i, inst := range instructions {
		oldToNew[inst.offset] = currentOffset
		instructions[i].offset = currentOffset

		def, _ := bytecode.Lookup(inst.opcode)
		size := 1
		for _, w := range def.OperandWidths {
			size += w
		}
		currentOffset += size
	}

	// Second pass: fix jump targets
	for i, inst := range instructions {
		switch inst.opcode {
		case bytecode.OpJump, bytecode.OpJumpIfFalse, bytecode.OpJumpIfTrue,
			bytecode.OpAnd, bytecode.OpOr, bytecode.OpUnwrapOrJump:
			if len(inst.operands) > 0 {
				oldTarget := inst.operands[0]
				if newTarget, ok := oldToNew[oldTarget]; ok {
					instructions[i].operands[0] = newTarget
				}
				// If target not found, it might be pointing to the end or an external location
				// Keep the original target in that case
			}
		}
	}

	return instructions
}
