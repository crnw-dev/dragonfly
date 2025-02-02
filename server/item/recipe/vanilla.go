package recipe

import (
	// Ensure all blocks and items are registered before trying to load vanilla recipes.
	_ "github.com/df-mc/dragonfly/server/block"
	_ "github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"

	_ "embed"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

var (
	//go:embed crafting_data.nbt
	vanillaCraftingData []byte
	//go:embed smithing_data.nbt
	vanillaSmithingData []byte
)

// shapedRecipe is a recipe that must be crafted in a specific shape.
type shapedRecipe struct {
	Input    inputItems  `nbt:"input"`
	Output   outputItems `nbt:"output"`
	Block    string      `nbt:"block"`
	Width    int32       `nbt:"width"`
	Height   int32       `nbt:"height"`
	Priority int32       `nbt:"priority"`
}

// shapelessRecipe is a recipe that may be crafted without a strict shape.
type shapelessRecipe struct {
	Input    inputItems  `nbt:"input"`
	Output   outputItems `nbt:"output"`
	Block    string      `nbt:"block"`
	Priority int32       `nbt:"priority"`
}

func init() {
	var craftingRecipes struct {
		Shaped    []shapedRecipe    `nbt:"shaped"`
		Shapeless []shapelessRecipe `nbt:"shapeless"`
	}
	if err := nbt.Unmarshal(vanillaCraftingData, &craftingRecipes); err != nil {
		panic(err)
	}

	var smithingRecipes []shapelessRecipe
	if err := nbt.Unmarshal(vanillaSmithingData, &smithingRecipes); err != nil {
		panic(err)
	}

	for _, s := range append(craftingRecipes.Shapeless, smithingRecipes...) {
		input, ok := s.Input.Stacks()
		output, okTwo := s.Output.Stacks()
		if !ok || !okTwo {
			// This can be expected to happen, as some recipes contain blocks or items that aren't currently implemented.
			continue
		}
		b, _ := world.BlockByName("minecraft:"+s.Block, nil)
		Register(Shapeless{recipe{
			input:    input,
			output:   output,
			block:    b,
			priority: uint32(s.Priority),
		}})
	}

	for _, s := range craftingRecipes.Shaped {
		input, ok := s.Input.Stacks()
		output, okTwo := s.Output.Stacks()
		if !ok || !okTwo {
			// This can be expected to happen - refer to the comment above.
			continue
		}
		b, _ := world.BlockByName("minecraft:"+s.Block, nil)
		Register(Shaped{
			shape: Shape{int(s.Width), int(s.Height)},
			recipe: recipe{
				input:    input,
				output:   output,
				block:    b,
				priority: uint32(s.Priority),
			},
		})
	}
}
