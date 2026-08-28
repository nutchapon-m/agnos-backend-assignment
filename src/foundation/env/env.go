package env

import (
	"strings"

	"github.com/spf13/viper"
)

type Env struct{}

func Load(path, file string) (Env, error) {
	part := strings.Split(file, ".")

	viper.SetConfigName(part[0])
	viper.SetConfigType(part[1])
	viper.AddConfigPath(path)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return Env{}, err
	}
	return Env{}, nil
}

func (Env) GetString(key string) string {
	return viper.GetString(key)
}

func (Env) GetInt(key string) int {
	return viper.GetInt(key)
}
