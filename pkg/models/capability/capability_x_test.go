package capability

import "testing"

func TestCapability_EnrichSortableFields(t *testing.T) {
	tests := []struct {
		name            string
		description     string
		parameters      []SwaggerParam
		wantSortDesc    string
		wantDescWithout string
	}{
		{
			name:        "normal case",
			description: "List users <sortable>name,email,created_at</sortable> API",
			parameters: []SwaggerParam{
				{Name: "limit", Description: "Page size"},
				{Name: "sort", Description: "Sort field"},
			},
			wantSortDesc:    "Sort field Sortable fields: name, email, created_at",
			wantDescWithout: "List users  API",
		},
		{
			name:        "sortable at start",
			description: "<sortable>id,name</sortable> Get all items",
			parameters: []SwaggerParam{
				{Name: "sort", Description: "Sort by"},
			},
			wantSortDesc:    "Sort by Sortable fields: id, name",
			wantDescWithout: " Get all items",
		},
		{
			name:        "sortable at end",
			description: "Get items <sortable>price,quantity</sortable>",
			parameters: []SwaggerParam{
				{Name: "sort", Description: ""},
			},
			wantSortDesc:    "Sortable fields: price, quantity",
			wantDescWithout: "Get items ",
		},
		{
			name:        "no sortable tag",
			description: "Just a normal description",
			parameters: []SwaggerParam{
				{Name: "sort", Description: "Sort field"},
			},
			wantSortDesc:    "Sort field",
			wantDescWithout: "Just a normal description",
		},
		{
			name:        "no sort parameter",
			description: "List users <sortable>name,email</sortable> API",
			parameters: []SwaggerParam{
				{Name: "limit", Description: "Page size"},
			},
			wantSortDesc:    "",
			wantDescWithout: "List users <sortable>name,email</sortable> API",
		},
		{
			name:        "unclosed sortable tag",
			description: "List users <sortable>name,email",
			parameters: []SwaggerParam{
				{Name: "sort", Description: "Sort field"},
			},
			wantSortDesc:    "Sort field",
			wantDescWithout: "List users <sortable>name,email",
		},
		{
			name:        "empty sortable content",
			description: "List users <sortable></sortable> API",
			parameters: []SwaggerParam{
				{Name: "sort", Description: ""},
			},
			wantSortDesc:    "Sortable fields: ",
			wantDescWithout: "List users  API",
		},
		{
			name:        "single field",
			description: "Get items <sortable>id</sortable>",
			parameters: []SwaggerParam{
				{Name: "sort", Description: "Sort by"},
			},
			wantSortDesc:    "Sort by Sortable fields: id",
			wantDescWithout: "Get items ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CapabilityBasic{
				Description: tt.description,
				Parameters:  tt.parameters,
			}
			c.EnrichSortableFields()

			var sortDesc string
			for _, p := range c.Parameters {
				if p.Name == "sort" {
					sortDesc = p.Description
					break
				}
			}
			if sortDesc != tt.wantSortDesc {
				t.Errorf("sort param description = %q, want %q", sortDesc, tt.wantSortDesc)
			}

			if c.Description != tt.wantDescWithout {
				t.Errorf("Description = %q, want %q", c.Description, tt.wantDescWithout)
			}
		})
	}
}
