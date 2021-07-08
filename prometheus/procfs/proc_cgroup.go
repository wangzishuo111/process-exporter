// Copyright 2020 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package procfs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/procfs/internal/util"
)

// Cgroup models one line from /proc/[pid]/cgroup. Each Cgroup struct describes the the placement of a PID inside a
// specific control hierarchy. The kernel has two cgroup APIs, v1 and v2. v1 has one hierarchy per available resource
// controller, while v2 has one unified hierarchy shared by all controllers. Regardless of v1 or v2, all hierarchies
// contain all running processes, so the question answerable with a Cgroup struct is 'where is this process in
// this hierarchy' (where==what path on the specific cgroupfs). By prefixing this path with the mount point of
// *this specific* hierarchy, you can locate the relevant pseudo-files needed to read/set the data for this PID
// in this hierarchy
//
// Also see http://man7.org/linux/man-pages/man7/cgroups.7.html
type Cgroup struct {
	// HierarchyID that can be matched to a named hierarchy using /proc/cgroups. Cgroups V2 only has one
	// hierarchy, so HierarchyID is always 0. For cgroups v1 this is a unique ID number
	HierarchyID int
	// Controllers using this hierarchy of processes. Controllers are also known as subsystems. For
	// Cgroups V2 this may be empty, as all active controllers use the same hierarchy
	Controllers []string
	// Path of this control group, relative to the mount point of the cgroupfs representing this specific
	// hierarchy
	Path string
	CgroupMemMax int64
}


// parseCgroupString parses each line of the /proc/[pid]/cgroup file
// Line format is hierarchyID:[controller1,controller2]:path
func parseCgroupString(cgroupStr string) (*Cgroup, error) {
	var err error

	fields := strings.Split(cgroupStr, ":")
	if len(fields) < 3 {
		return nil, fmt.Errorf("at least 3 fields required, found %d fields in cgroup string: %s", len(fields), cgroupStr)
	}
	cgroup := &Cgroup{
		Path:        fields[2],
		Controllers: nil,
	}
	if fields[1] == "memory" {
		cgroupfile := "/sys/fs/cgroup/memory" + fields[2]
		myfile := cgroupfile + "/memory.limit_in_bytes"
		_, err := os.Stat(myfile)
		if err == nil {
			//data, _ := ioutil.ReadFile(myfile)
			data, _ := util.ReadFileNoStat(fmt.Sprintf("%v", myfile))
			trimdata := strings.TrimSpace(string(data))
			CgroupMemMax, _ := strconv.ParseInt(trimdata, 10, 64)
			cgroup.CgroupMemMax = CgroupMemMax
		}
	}
	cgroup.HierarchyID, err = strconv.Atoi(fields[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse hierarchy ID")
	}
	if fields[1] != "" {
		ssNames := strings.Split(fields[1], ",")
		cgroup.Controllers = append(cgroup.Controllers, ssNames...)
	}
	return cgroup, nil
}

// parseCgroups reads each line of the /proc/[pid]/cgroup file
func parseCgroups(data []byte) ([]Cgroup, error) {
	var cgroups []Cgroup
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		mountString := scanner.Text()
		parsedMounts, err := parseCgroupString(mountString)
		if err != nil {
			return nil, err
		}
		if parsedMounts.Controllers[0] != "memory" {
			continue
		}
		cgroups = append(cgroups, *parsedMounts)
	}

	err := scanner.Err()
	return cgroups, err
}

// Cgroups reads from /proc/<pid>/cgroups and returns a []*Cgroup struct locating this PID in each process
// control hierarchy running on this system. On every system (v1 and v2), all hierarchies contain all processes,
// so the len of the returned struct is equal to the number of active hierarchies on this system
func (p Proc) Cgroups() ([]Cgroup, error) {
	data, err := util.ReadFileNoStat(fmt.Sprintf("/proc/%d/cgroup", p.PID))
	if err != nil {
		return nil, err
	}
	return parseCgroups(data)
}


//func (p Proc) MyCgroups() (Cgroup, error) {
//	var clist []Cgroup
//	data, err := util.ReadFileNoStat(fmt.Sprintf("/proc/%d/cgroup", p.PID))
//	if err != nil {
//		return Cgroup{}, err
//	}
//	scanner := bufio.NewScanner(bytes.NewReader(data))
//	for scanner.Scan() {
//		mountString := scanner.Text()
//		parsedMounts, err := parseCgroupString(mountString)
//		if err != nil {
//			return Cgroup{}, err
//		}
//		clist = append(clist, *parsedMounts)
//		cgroups := *parsedMounts
//	}
//	return Cgroup{}, err
//}

func (p Proc) NewCgroup() ([]Cgroup, error) {
   //aa, _ := p.Cgroups()
   return p.Cgroups()
}


//func (c Cgroup) CgroupMemMax() int64 {
//	return s.VSize
//}