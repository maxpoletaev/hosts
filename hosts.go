package main

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// DebugFlag enables output to console instead of file.
var DebugFlag bool

// HostsFile is a path to the hosts file.
var HostsFile string

// Host describes the row in hosts file.
type Host struct {
	IP        string
	Hostnames []string
}

// ReadHosts parses the /etc/hosts file.
func ReadHosts(hosts *[]interface{}) {
	contents, err := ioutil.ReadFile(HostsFile)
	if err != nil {
		log.Fatal(err)
	}

	rows := strings.Split(string(contents), "\n")

	for _, row := range rows {
		row = strings.TrimSpace(row)

		// skip comments and empty lines
		if strings.HasPrefix(row, "#") || row == "" {
			*hosts = append(*hosts, row)
			continue
		}

		// skip invalid records
		fields := strings.Fields(row)
		if len(fields) < 2 {
			*hosts = append(*hosts, row)
			continue
		}

		ip := fields[0]
		hostnames := fields[1:]
		host := Host{ip, hostnames}
		*hosts = append(*hosts, host)
	}
}

// WriteHosts updates the /etc/hosts file.
func WriteHosts(hosts []interface{}) {
	renderedHosts := RenderHosts(hosts)

	if DebugFlag {
		fmt.Print(renderedHosts)
		return
	}

	contents := []byte(renderedHosts)
	err := ioutil.WriteFile(HostsFile, contents, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

// RenderHosts transforms code representation of the hosts file to string.
func RenderHosts(hosts []interface{}) string {
	var buf bytes.Buffer

	for _, host := range hosts {
		switch v := host.(type) {

		case string:
			buf.WriteString(v)

		case Host:
			hostnames := strings.Join(v.Hostnames, " ")
			row := v.IP + "\t" + hostnames + "\n"
			buf.WriteString(row)
		}
	}

	return buf.String()
}

// RenderHostsWithoutComments transforms code representation of the hosts
// file to string, but ignores empty and commented (that starts with #) strings.
func RenderHostsWithoutComments(hosts []interface{}) string {
	var buf bytes.Buffer

	for _, host := range hosts {
		if v, ok := host.(Host); ok {
			hostnames := strings.Join(v.Hostnames, " ")
			row := v.IP + "\t" + hostnames + "\n"
			buf.WriteString(row)
		}
	}

	return buf.String()
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

func removeHostname(hostnames []string, remove string) []string {
	removeIndex := -1
	for i, hostname := range hostnames {
		if hostname == remove {
			removeIndex = i
		}
	}
	if removeIndex != -1 {
		hostnames = append(hostnames[:removeIndex], hostnames[removeIndex+1:]...)
	}
	return hostnames
}

var rootCmd = &cobra.Command{
	Use: "hosts",
}

var cmdAddHost = &cobra.Command{
	Use:  "add ip hostname [hostname ...]",
	Short: "Add host to the hosts file",
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var hosts []interface{}
		ReadHosts(&hosts)

		ip := args[0]
		hostnames := args[1:]

		updated := false
		for i, host := range hosts {
			if host, ok := host.(Host); ok {
				if host.IP == ip {
					hostnames = append(host.Hostnames, hostnames...)
					host.Hostnames = uniqueStrings(hostnames)
					hosts[i] = host
					updated = true
					break
				}
			}
		}

		if !updated {
			host := Host{ip, uniqueStrings(hostnames)}
			hosts = append(hosts, host)
		}

		WriteHosts(hosts)
	},
}

var cmdResolve = &cobra.Command{
	Use:  "resolve ip",
	Short: "Resolve hostname to IP address",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var hosts []interface{}
		ReadHosts(&hosts)

		searchHostname := args[0]

		for _, host := range hosts {
			if host, ok := host.(Host); ok {
				for _, hostname := range host.Hostnames {
					if hostname == searchHostname {
						fmt.Println(host.IP)
						return
					}
				}
			}
		}
	},
}

var cmdList = &cobra.Command{
	Use:  "list",
	Short: "List all hosts",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var hosts []interface{}
		ReadHosts(&hosts)

		renderedHosts := RenderHostsWithoutComments(hosts)
		fmt.Print(renderedHosts)
	},
}

var cmdRemoveIP = &cobra.Command{
	Use:  "rmip ip",
	Short: "Remove IP address from the hosts file.",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var hosts []interface{}
		ReadHosts(&hosts)

		ip := args[0]
		for i := len(hosts) - 1; i >= 0; i-- {
			if host, ok := hosts[i].(Host); ok {
				if host.IP == ip {
					hosts = append(hosts[:i], hosts[i+1:])
				}
			}
		}

		WriteHosts(hosts)
	},
}

var cmdRemoveHost = &cobra.Command{
	Use:  "rmhost hostname [hostname ...]",
	Short: "Remove hostname from the hosts file",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var hosts []interface{}
		ReadHosts(&hosts)

		hostnames := args
		for i := len(hosts) - 1; i >= 0; i-- {
			if host, ok := hosts[i].(Host); ok {
				for _, hostname := range hostnames {
					host.Hostnames = removeHostname(host.Hostnames, hostname)
				}
				if len(host.Hostnames) > 0 {
					hosts[i] = host
				} else {
					hosts = append(hosts[:i], hosts[i+1:]...)
				}
			}
		}

		WriteHosts(hosts)
	},
}

func main() {
	rootCmd.PersistentFlags().BoolVarP(&DebugFlag, "debug", "d", false, "print output to console instead of file")
	rootCmd.PersistentFlags().StringVarP(&HostsFile, "file", "f", "/etc/hosts", "path to the hosts file")

	rootCmd.AddCommand(cmdList)
	rootCmd.AddCommand(cmdResolve)
	rootCmd.AddCommand(cmdAddHost)
	rootCmd.AddCommand(cmdRemoveIP)
	rootCmd.AddCommand(cmdRemoveHost)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
