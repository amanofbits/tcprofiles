# TCProfiles

TLP Config Profiles (not affiliated with tlp) - A small tool that allows to use different profiles of [TLP](https://github.com/linrunner/TLP) configuration

## Why?

TLP is a great tool, but I found myself in situation where I need to change my configuration often, and I haven't found how to use tlp with multiple configs (profiles). It allows to set different settings for AC and battery, but as far as I understand (maybe I missed something while searching?), it ignores the fact that "AC" is not just AC anymore. Laptops are now powered via USB type-c and can make use of PD powerbanks which are still treated as an AC adapter by the laptop, while effectively being an external battery.

TCProfiles was created to work around this problem by defining and then quickly switching configuration profiles.

Of course you can have multiple config files and replace them, or have them all in tlp.d and rename them to change priorities,
but it looks impractical to me if has to be done often.
This is why TCProfiles was created. It allows to create one single template config with different profiles, and apply them as necessary,
using just a few commands. Yes, this probably can be done with single small bash script but I just find Go easier to
work with.

## Building

You can just grab the amd64 linux release build from releases section, or build from source.

if you have Golang installed, then simply grab the repo, and run
```
go build .
```
within the folder. App has no dependencies.

## Usage

### First run

If you run this tool the first time, you need to create template first. The template is a simple text file, like tlp.conf
with added ini/toml-like sections that correspond to profiles. It can be created separately with the name `tcptemplate.txt`,
or by running

```
./tcprofiles template
```

Then you need to add all settings and profiles according to expected usage scenarios to the template and save it.

### Selecting profiles

After template is properly configured, profiles can easily be switched by executing

```
./tcprofiles use <profile1>[ <profile2> ...] | sudo tee /etc/tlp.d/50-config.conf
```

You can change the destination file to whatever you see fit, or use the usual linux redirection techniques. The tool outputs only resulting configuration to STDOUT and other information to the STDERR.

You can specify 1 or more profiles, they will be applied one by one left to right, duplicate settings from last override such from first.

You can specify 'default' only as the single profile, or only as the first one (which is unnecessary because it is always prepended).

Optionally, you can validate the output by simply executing

```
./tcprofiles use <profile1>[ <profile2> ...]
```

and looking to the output.

### Applying changes

Don't forget to run
```
tlp start
```

after profile selection to make changes work. Remember that probably not all settings are applied immediately, please consult
tlp's documentation for details.

## Notes
- I hope I didn't overlook something obvious while searching. Didn't want to make false claims about TLP, just wanted to make a useful tool.
- Feel free to use and file issues.
- it's easy to integrate creation of resulting config file and applying changes into the tool to make process even easier but I don't really like the idea to make this `sudo`-involved process opaque.
