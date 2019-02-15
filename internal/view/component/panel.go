package component

import (
	"encoding/json"
)

// PanelConfig is the contents of a Panel
type PanelConfig struct {
	Content  ViewComponent `json:"content"`
	Position PanelPosition `json:"position"`
}

func (t *PanelConfig) UnmarshalJSON(data []byte) error {
	x := struct {
		Position PanelPosition
		Content  typedObject
	}{}

	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}

	t.Position = x.Position

	var err error
	t.Content, err = x.Content.ToViewComponent()
	if err != nil {
		return err
	}

	return nil
}

// PanelPosition represents the relative location and size of a panel within a grid
type PanelPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// Panel contains other ViewComponents
type Panel struct {
	base
	Config PanelConfig `json:"config"`
}

// NewPanel creates a panel component
func NewPanel(title string, content ViewComponent) *Panel {
	return &Panel{
		base: newBase(typePanel, TitleFromString(title)),
		Config: PanelConfig{
			Content: content,
		},
	}
}

// Position sets the position for the panel in a grid.
func (t *Panel) Position(x, y, w, h int) {
	t.Config.Position.X = x
	t.Config.Position.Y = y
	t.Config.Position.W = w
	t.Config.Position.H = h
}

// GetMetadata accesses the components metadata. Implements ViewComponent.
func (t *Panel) GetMetadata() Metadata {
	return t.Metadata
}

type panelMarshal Panel

// MarshalJSON implements json.Marshaler
func (t *Panel) MarshalJSON() ([]byte, error) {
	m := panelMarshal(*t)
	m.Metadata.Type = typePanel
	return json.Marshal(&m)
}
