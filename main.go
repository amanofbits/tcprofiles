// Copyright (c) 2024, amanofbits

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

const (
	templateFile       = "./tctemplate.txt"
	defaultProfileName = "default"
	template           = `# Profiles are defined as ini/toml sections, e.g. [profile_name]
# Values before any profile defined belong to default profile, they will be used if not overridden in specific profile.
# Lines starting with '#' are comments (won't go into produced file)
#
# Default profile is usually a baseline for a day-to-day device usage.
# Specific profiles exist to override defaults for specific situations
# (like when "AC" is actually a powerbank, and needs to be treated like battery)
#
# You can have specific profiles for AC and BAT and combine them in different ways,
# tlp documentation can be fount at https://linrunner.de/tlp/settings/
#
# Example:
# [default]
# TLP_ENABLE=0
# ... etc
#
# [ac_powerbank]
# TLP_ENABLE=1
# ... etc
`
)

var errNoArguments = errors.New("no arguments specified")

func main() {
	selected, err := parseInput()
	if err != nil {
		if !errors.Is(err, errNoArguments) {
			logToErr("%v\n\n", err)
		}
		if errors.Is(err, errNoArguments) {
			template, err := parseTemplate()
			if errors.Is(err, os.ErrNotExist) {
				logToErr("Template does not exist\n")
			} else if err == nil {
				logToErr("Profiles found in template: %s\n", strings.Join(getProfiles(template), ", "))
			}
		}
		printUsage()
		os.Exit(1)
	}
	logToErr("Profiles selected: %s;\n", strings.Join(selected, ", "))

	template, err := parseTemplate()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logToErr("Error: template file does not exist. Please create one\n")
			printUsage()
			os.Exit(1)
		}
		logToErr("Template error: %v\n", err)
		os.Exit(1)
	}
	logToErr("Profiles found in template: %s\n", strings.Join(getProfiles(template), ", "))

	config := strings.Builder{}
	fillConfig(&config, template, selected)

	logToErr("Output:\n")

	logToOut("%s\n", config.String())
}

func logToErr(msg string, args ...any) {
	s := fmt.Sprintf(msg, args...)
	if len(s) != 0 {
		sr := []rune(s)
		sr[0] = unicode.ToUpper(sr[0])
		s = string(sr)
	}
	fmt.Fprintf(os.Stderr, "%s", s)
}

func logToOut(msg string, args ...any) {
	fmt.Fprintf(os.Stdout, msg, args...)
}

func printUsage() {
	tool := filepath.Base(os.Args[0])
	logToErr(`
Usage:
	This tool allows to create tlp config text using profiles from a template.
	1) Generate template file '%s' (won't be overwritten if
	   already exist)
		./%s template
	2) Add profiles with tlp settings to the template and save the file.
	3) Select profile[s] and validate output
		./%s use <profile1>[ <profile2>[ <profileN>]]
	4) Write output to tlp config	
		./%s use default | sudo tee /etc/tlp.d/50-config.conf

	Remember that you need to run tlp start to apply changes.
	Or you can run it all in one line:
		./%s use default | sudo tee /etc/tlp.d/50-config.conf && sudo tlp start

	You can specify one or more profiles, they will be applied one by one left
	to right, duplicate settings from last overrides such from first.

	You can specify 'default' only as the single, or the first (which is
	unnecessary) profile.
`, filepath.Base(templateFile), tool, tool, tool, tool)
}

type kv struct{ key, value string }
type sectionLine struct {
	profile string
	setting kv
}

func getProfiles(sls []sectionLine) []string {
	ps := make(map[string]struct{}, 0)
	for _, sl := range sls {
		ps[sl.profile] = struct{}{}
	}

	delete(ps, defaultProfileName)

	var psArr []string
	for p := range ps {
		psArr = append(psArr, p)
	}
	slices.Sort(psArr)

	return append([]string{defaultProfileName}, psArr...)
}

var sectionRegex = regexp.MustCompile(`^\[.*\]$`)
var validSectionNameRegex = regexp.MustCompile(`^[\w\d]+$`)
var keyValRegex = regexp.MustCompile(`^([\w]+?)=(.+)$`)

func parseTemplate() (lines []sectionLine, err error) {
	f, err := os.Open(templateFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bf := bufio.NewReader(f)

	curProfile := defaultProfileName
	lineNum := 0
	for {
		lineNum++
		line, err := bf.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("template read line %d error: %v", lineNum, err)
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		if sectionRegex.MatchString(line) {
			p := line[1 : len(line)-1]
			if !validSectionNameRegex.MatchString(p) {
				return nil, fmt.Errorf("malformed section name %q at line %d. Latin letters, digits and underscores only",
					p, lineNum)
			}
			curProfile = p
		} else {
			kvMatches := keyValRegex.FindStringSubmatch(line)
			if len(kvMatches) < 3 {
				return nil, fmt.Errorf("malformed template line %d: %s", lineNum, line)
			}
			lines = append(lines, sectionLine{
				profile: curProfile,
				setting: kv{
					key:   kvMatches[1],
					value: kvMatches[2],
				},
			})
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}
	return lines, nil
}

func lastIndex[S ~[]E, E comparable](s S, v E) int {
	for i := len(s) - 1; i != 0; i-- {
		if v == s[i] {
			return i
		}
	}
	return -1
}

func parseInput() (profiles []string, err error) {
	inputs := os.Args[1:]
	if len(inputs) == 0 {
		return nil, errNoArguments
	}

	h := flag.Bool("help", false, "")
	hs := flag.Bool("h", false, "")
	flag.Parse()

	if *h || *hs {
		return nil, errNoArguments
	}

	if inputs[0] == "template" {
		createTemplateFile()
		os.Exit(0)
	}

	if inputs[0] != "use" {
		return nil, fmt.Errorf("unknown command %q", inputs[0])
	}

	inputs = inputs[1:]

	if len(inputs) == 0 {
		return nil, fmt.Errorf("no profile[s] selected")
	}

	for i := 0; i < len(inputs); i++ {
		profiles = append(profiles, inputs[i])
	}

	if len(profiles) > 0 && lastIndex(profiles, defaultProfileName) > 0 {
		return nil, fmt.Errorf("default profile must be the only, or the first of many selections.\n\tGot %q",
			strings.Join(profiles, ","))
	}

	return profiles, nil
}

func createTemplateFile() {
	f, err := os.OpenFile(templateFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			logToErr("Error creating template file %q: already exists\n", templateFile)
		} else {
			logToErr("Error creating template: %v\n", err)
		}
		os.Exit(1)
	}
	defer f.Close()

	f.WriteString(template)
}

func fillConfig(config *strings.Builder, template []sectionLine, selected []string) {
	fmt.Fprintf(config, "# Generated by tcprofiles command\n\n")

	if selected[0] == defaultProfileName {
		selected = selected[1:]
	}

	templateCache := slices.Clone(template)
	settings := make([]kv, 0)
	settingIdx := make(map[string]int)

	for _, profile := range append([]string{defaultProfileName}, selected...) {
		for i := 0; i < len(templateCache); i++ {
			if templateCache[i].profile != profile {
				continue
			}
			settings = append(settings, templateCache[i].setting)
			settingIdx[templateCache[i].setting.key] = len(settings) - 1
			templateCache = append(templateCache[:i], templateCache[i+1:]...)
			i--
		}
	}

	for idx, setting := range settings {
		if settingIdx[setting.key] == idx {
			fmt.Fprintf(config, "%s=%s\n", setting.key, setting.value)
		}
	}
}
