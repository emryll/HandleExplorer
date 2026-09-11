package cli

import (
	"HandleExplorer/handles"
	"HandleExplorer/handles/registry"
	"HandleExplorer/nt"
	"HandleExplorer/store"
	"HandleExplorer/utils"
	"flag"
	"strconv"
	"strings"
)

//*===============================[ Flag parsing ]=====================================

// Parse command-line flags for the "find" command.
func parseFindFlags(flags []string) *handles.HandleFilter {
	var filter handles.HandleFilter
	fs := flag.NewFlagSet("find", flag.ExitOnError)

	var (
		rawTargets string
		rawObjType string
		rawAccess  string
		rawNames   string
	)

	fs.StringVar(&rawTargets, "p", "", "process filter")
	fs.StringVar(&rawObjType, "o", "", "object type filter")
	fs.StringVar(&rawNames, "n", "", "object name filter")
	fs.StringVar(&rawAccess, "a", "", "handle access filter")
	fs.Parse(flags)

	filter.Pids = parsePsTargetString(rawTargets)
	filter.Access = utils.ParseAccessString(rawAccess)

	for _, objType := range strings.Split(rawObjType, ",") {
		objType = strings.TrimSpace(objType)
		if enum := nt.GetTypeIdentifier(objType); enum != nt.OBJ_TYPE_UNKNOWN {
			filter.ObjType = append(filter.ObjType, enum)
		}
	}

	for _, name := range strings.Split(rawNames, ",") {
		name = strings.TrimSpace(name)
		filter.Names = append(filter.Names, name)
	}

	return &filter
}

func parsePsTargetString(targets string) []uint32 {
	var pids []uint32
	tokens := strings.Split(targets, ",")

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if pid, err := strconv.Atoi(token); err == nil {
			pids = append(pids, uint32(pid))
		} else {
			pids = append(pids, store.PsTable.FindProcesses(token)...)
		}
	}
	return pids
}

func parseClustersFlags(flags []string) registry.ClusterFilter {
	var (
		filter      registry.ClusterFilter
		objTypeName string
	)
	cf := flag.NewFlagSet("clusters", flag.ExitOnError)

	cf.IntVar(&filter.MinSize, "m", 0, "minimum cluster size")
	cf.StringVar(&filter.ObjName, "n", "", "object name")
	cf.StringVar(&objTypeName, "o", "", "object type")

	cf.Parse(flags)
	filter.ObjType = nt.GetTypeIdentifier(objTypeName)
	return filter
}
