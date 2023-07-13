package flags

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func Bool(fs *pflag.FlagSet, key, name, usage string) {
	defValue := viper.GetBool(key)
	fs.Bool(name, defValue, usage)
	viper.BindPFlag(key, fs.Lookup(name))
}

func BoolP(fs *pflag.FlagSet, key, name, shorthand, usage string) {
	defValue := viper.GetBool(key)
	fs.BoolP(name, shorthand, defValue, usage)
	viper.BindPFlag(key, fs.Lookup(name))
}

func String(fs *pflag.FlagSet, key, name, usage string) {
	defValue := viper.GetString(key)
	fs.String(name, defValue, usage)
	viper.BindPFlag(key, fs.Lookup(name))
}

func StringP(fs *pflag.FlagSet, key, name, shorthand, usage string) {
	defValue := viper.GetString(key)
	fs.StringP(name, shorthand, defValue, usage)
	viper.BindPFlag(key, fs.Lookup(name))
}

func StringSlice(fs *pflag.FlagSet, key, name, usage string) {
	defValue := viper.GetStringSlice(key)
	fs.StringSlice(name, defValue, usage)
	viper.BindPFlag(key, fs.Lookup(name))
}

func StringSliceP(fs *pflag.FlagSet, key, name, shorthand, usage string) {
	defValue := viper.GetStringSlice(key)
	fs.StringSliceP(name, shorthand, defValue, usage)
	viper.BindPFlag(key, fs.Lookup(name))
}

func Int(fs *pflag.FlagSet, key, name, usage string) {
	defValue := viper.GetInt(key)
	fs.Int(name, defValue, usage)
	viper.BindPFlag(key, fs.Lookup(name))
}

func IntP(fs *pflag.FlagSet, key, name, shorthand, usage string) {
	defValue := viper.GetInt(key)
	fs.IntP(name, shorthand, defValue, usage)
	viper.BindPFlag(key, fs.Lookup(name))
}

func Duration(fs *pflag.FlagSet, key, name, usage string) {
	defValue := viper.GetDuration(key)
	fs.Duration(name, defValue, usage)
	viper.BindPFlag(key, fs.Lookup(name))
}
