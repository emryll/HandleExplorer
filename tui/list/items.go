package tlist

import (
	"fmt"
	"path/filepath"
)

type ListItem interface {
	Columns() []Column
	Fields() []string
	Key() string
	// Used for newPickerModel(...)
	Title() string
	Subtitle() string
	Noun() string
}

const (
	requiredProcessFields = 2
	requiredHandleFields  = 2
	requiredClusterFields = 2
)

//*===================[ Process ]===================

type Process struct {
	Pid         uint32
	Path        string
	ParentPid   uint32
	ParentPath  string
	SigStatus   uint32
	HandleCount int
}

func (p Process) Columns() []Column {
	return []Column{
		{Title: "Handles"},
		{Title: "PID", Highlight: true},
		{Title: "Path", Highlight: true},
		{Title: "Parent", Right: true},
	}
}

func (p Process) Fields() []string {
	parent := fmt.Sprintf("PID %d", p.ParentPid)
	if p.ParentPath != "" {
		parent += fmt.Sprintf(" (%s)", filepath.Base(p.ParentPath))
	}

	return []string{
		fmt.Sprint(p.HandleCount),
		fmt.Sprint(p.Pid),
		p.Path,
		parent,
	}
}

func (p Process) Key() string { return fmt.Sprintf("%d", p.Pid) }

// RightStages implements StagedField for the Parent column: full
// "PID x (name)" if there's room, then "PID x (...)" to indicate a
// name exists without showing it, then just "PID x", then hidden.
func (p Process) RightStages() []string {
	if p.ParentPid == 0 {
		return []string{""}
	}

	if p.ParentPath == "" {
		return []string{fmt.Sprintf("PID %d", p.ParentPid)}
	}

	return []string{
		fmt.Sprintf("PID %d (%s)", p.ParentPid, p.ParentPath),
		fmt.Sprintf("PID %d (...)", p.ParentPid),
		fmt.Sprintf("PID %d", p.ParentPid),
	}
}

// Used for polymorphic list (newPickerModel)
func (p Process) Title() string {
	return "Process"
}

// Used for polymorphic list (newPickerModel)
func (p Process) Subtitle() string {
	return "Process search results"
}

// Used for polymorphic list (newPickerModel)
func (p Process) Noun() string {
	return "processes"
}

//*================[ Handle ]===================

type HandleEntry struct {
	AccessingPid  uint32
	AccessingPath string
	AccessMask    string
	ObjectType    string
	ObjectName    string
}

func (h HandleEntry) Columns() []Column {
	return []Column{
		{Title: "Type", Highlight: true},
		{Title: "Name"},
		{Title: "Accessing Process", Highlight: true, Right: true},
		{Title: "Access", Right: true},
	}
}

func (h HandleEntry) Fields() []string {
	return []string{
		h.ObjectType,
		orDash(h.ObjectName),
		fmt.Sprintf("%d", h.AccessingPid),
		h.AccessMask,
	}
}

func (h HandleEntry) Key() string {
	return fmt.Sprintf("%d:%s:%s:%s", h.AccessingPid, h.ObjectType, h.ObjectName, h.AccessMask)
}

// Used for polymorphic list (newPickerModel)
func (h HandleEntry) Title() string {
	return "Handle"
}

// Used for polymorphic list (newPickerModel)
func (h HandleEntry) Subtitle() string {
	return "Handle search results"
}

// Used for polymorphic list (newPickerModel)
func (h HandleEntry) Noun() string {
	return "handles"
}

//*==================[ Cluster ]=====================

type Cluster struct {
	ObjType string
	ObjName string
	Size    int
}

func (c Cluster) Columns() []Column {
	return []Column{
		{Title: "Type", Highlight: true},
		{Title: "Name"},
		{Title: "Size"},
	}
}

func (c Cluster) Fields() []string {
	return []string{
		c.ObjType,
		orDash(c.ObjName),
		fmt.Sprintf("%d in cluster", c.Size),
	}
}

func (c Cluster) Key() string {
	return fmt.Sprintf("%s:%s", c.ObjType, c.ObjName)
}

// Used for polymorphic list (newPickerModel)
func (c Cluster) Title() string {
	return "Cluster"
}

// Used for polymorphic list (newPickerModel)
func (c Cluster) Subtitle() string {
	return "Cluster search results"
}

// Used for polymorphic list (newPickerModel)
func (c Cluster) Noun() string {
	return "clusters"
}
