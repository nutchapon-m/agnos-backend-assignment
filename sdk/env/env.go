package env

import (
	"strings"

	"github.com/spf13/viper"
)

func Load(path, file string) error {
	part := strings.Split(file, ".")

	viper.SetConfigName(part[0])
	viper.SetConfigType(part[1])
	viper.AddConfigPath(path)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	return nil
}
