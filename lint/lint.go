package lint

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Severity int

const (
	Success Severity = iota
	Exclude
	Warning
	Error
	Quiet

	Prometheus = "prometheus"
)

// Target is a deliberately incomplete representation of the Dashboard -> Template type in grafana.
// The properties which are extracted from JSON are only those used for linting purposes.
type Template struct {
	Name       string           `json:"name"`
	Label      string           `json:"label"`
	Type       string           `json:"type"`
	Query      string           `json:"query"`
	Datasource Datasource       `json:"datasource"`
	Multi      bool             `json:"multi"`
	AllValue   string           `json:"allValue"`
	Current    TemplateValue    `json:"current"`
	Options    []TemplateOption `json:"options"`
	Refresh    int              `json:"refresh"`
	// If you add properties here don't forget to add them to the raw struct, and assign them from raw to actual in UnmarshalJSON below!
}

type TemplateValue struct {
	Text  string `json:"text"`
	Value string `json:"value"`
}

type TemplateOption struct {
	TemplateValue
	Selected bool `json:"selected"`
}

func (t *Template) UnmarshalJSON(buf []byte) error {
	var raw struct {
		Name       string           `json:"name"`
		Label      string           `json:"label"`
		Type       string           `json:"type"`
		Query      interface{}      `json:"query"`
		Datasource Datasource       `json:"datasource"`
		Multi      bool             `json:"multi"`
		AllValue   string           `json:"allValue"`
		Current    TemplateValue    `json:"current"`
		Options    []TemplateOption `json:"options"`
		Refresh    int              `json:"refresh"`
	}

	if err := json.Unmarshal(buf, &raw); err != nil {
		return err
	}

	t.Name = raw.Name
	t.Label = raw.Label
	t.Type = raw.Type
	t.Datasource = raw.Datasource
	t.Multi = raw.Multi
	t.AllValue = raw.AllValue
	t.Current = raw.Current
	t.Options = raw.Options
	t.Refresh = raw.Refresh

	// the 'adhoc' and 'custom' variable type does not have a field `Query`, so we can't perform these checks
	if t.Type != "adhoc" && t.Type != "custom" {
		switch v := raw.Query.(type) {
		case string:
			t.Query = v
		case map[string]interface{}:
			t.Query = v["query"].(string)
		default:
			return fmt.Errorf("invalid type for field 'query': %v", v)
		}
	}

	return nil
}

func (t *TemplateValue) UnmarshalJSON(buf []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(buf, &raw); err != nil {
		return err
	}

	var txt, val interface{}

	txt, ok := raw["text"]
	if ok {
		switch tt := txt.(type) {
		case string:
			t.Text = txt.(string)
		case []interface{}:
			t.Text = txt.([]interface{})[0].(string)
		default:
			return fmt.Errorf("invalid type for field 'text': %v", tt)
		}
	}

	val, ok = raw["value"]
	if ok {
		switch vt := val.(type) {
		case string:
			t.Value = val.(string)
		case []interface{}:
			t.Value = val.([]interface{})[0].(string)
		default:
			return fmt.Errorf("invalid type for field 'value': %v", vt)
		}
	}

	return nil
}

type Datasource string

func (d *Datasource) UnmarshalJSON(buf []byte) error {
	var raw interface{}
	if err := json.Unmarshal(buf, &raw); err != nil {
		return err
	}

	switch v := raw.(type) {
	case nil:
		*d = ""
	case string:
		*d = Datasource(v)
	case map[string]interface{}:
		uid, ok := v["uid"]
		if !ok {
			return fmt.Errorf("invalid type for field 'datasource': missing uid field")
		}
		uidStr, ok := uid.(string)
		if !ok {
			return fmt.Errorf("invalid type for field 'datasource': invalid uid field type, should be string")
		}
		*d = Datasource(uidStr)
	default:
		return fmt.Errorf("invalid type for field 'datasource': %v", v)
	}

	return nil
}

// Target is a deliberately incomplete representation of the Dashboard -> Panel -> Target type in grafana.
// The properties which are extracted from JSON are only those used for linting purposes.
type Target struct {
	Idx          int    // This is the only (best?) way to uniquely identify a target, it is set by
	Expr         string `json:"expr,omitempty"`
	PanelId      int    `json:"panelId,omitempty"`
	RefId        string `json:"refId,omitempty"`
	LegendFormat string `json:"legendFormat,omitempty"`
}

// Panel is a deliberately incomplete representation of the Dashboard -> Panel type in grafana.
// The properties which are extracted from JSON are only those used for linting purposes.
type Panel struct {
	Id          int         `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Targets     []Target    `json:"targets,omitempty"`
	Datasource  Datasource  `json:"datasource"`
	Type        string      `json:"type"`
	Panels      []Panel     `json:"panels,omitempty"`
	FieldConfig FieldConfig `json:"fieldConfig"`
}

type FieldConfig struct {
	Defaults  Defaults   `json:"defaults"`
	Overrides []Override `json:"overrides"`
}

type Override struct {
	OverrideProperties []OverrideProperty `json:"properties"`
}

type OverrideProperty struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

func (o *OverrideProperty) UnmarshalJSON(buf []byte) error {
	// An override value can be of type string or int
	// This function detects type mismatch and uses an int type for Value
	var raw struct {
		Id    string `json:"id"`
		Value string `json:"value"`
	}

	if err := json.Unmarshal(buf, &raw); err != nil {
		var raw struct {
			Id    string `json:"id"`
			Value int    `json:"value"`
		}
		if err := json.Unmarshal(buf, &raw); err != nil {
			// Overrirde can have varying different types int, string and arrays
			// Currently only units are being checked from overrides so returning nil in case of unhandled types
			return nil
		}
	}

	return nil
}

type Defaults struct {
	Unit string `json:"unit"`
}

// GetPanels returns the all panels nested inside the panel (inc the current panel)
func (p *Panel) GetPanels() []Panel {
	panels := []Panel{*p}
	for _, panel := range p.Panels {
		panels = append(panels, panel.GetPanels()...)
	}
	return panels
}

// Row is a deliberately incomplete representation of the Dashboard -> Row type in grafana.
// The properties which are extracted from JSON are only those used for linting purposes.
type Row struct {
	Panels []Panel `json:"panels,omitempty"`
}

// GetPanels returns the all panels nested inside the row
func (r *Row) GetPanels() []Panel {
	var panels []Panel
	for _, panel := range r.Panels {
		panels = append(panels, panel.GetPanels()...)
	}
	return panels
}

// Dashboard is a deliberately incomplete representation of the Dashboard type in grafana.
// The properties which are extracted from JSON are only those used for linting purposes.
type Dashboard struct {
	Title      string `json:"title,omitempty"`
	Templating struct {
		List []Template `json:"list"`
	} `json:"templating"`
	Rows   []Row   `json:"rows,omitempty"`
	Panels []Panel `json:"panels,omitempty"`
}

// GetPanels returns the all panels whether they are nested in the (now deprecated) "rows" property or
// in the top level "panels" property. This also monkeypatches Target.Idx into each panel which is used
// to uniquely identify panel targets while linting.
func (d *Dashboard) GetPanels() []Panel {
	var p []Panel
	for _, row := range d.Rows {
		p = append(p, row.GetPanels()...)
	}
	for _, panel := range d.Panels {
		p = append(p, panel.GetPanels()...)
	}
	for pi, pa := range p {
		for ti := range pa.Targets {
			p[pi].Targets[ti].Idx = ti
		}
	}
	return p
}

// GetTemplateByType returns all dashboard templates which match the provided type. Type comparison
// is case insensitive as it uses strings.EqualFold()
func (d *Dashboard) GetTemplateByType(t string) []Template {
	var retval []Template
	for _, templ := range d.Templating.List {
		if strings.EqualFold(templ.Type, t) {
			retval = append(retval, templ)
		}
	}
	return retval
}

func NewDashboard(buf []byte) (Dashboard, error) {
	var dash Dashboard
	if err := json.Unmarshal(buf, &dash); err != nil {
		return dash, err
	}
	return dash, nil
}
