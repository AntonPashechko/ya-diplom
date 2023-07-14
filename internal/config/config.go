package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/AntonPashechko/ya-diplom/internal/logger"
)

type Config struct {
	Endpoint       string
	DataBaseDNS    string
	AccrualAddress string
	JWTKey         []byte        //Ключ для создания/проверки jwt для авторизации
	JWTDuration    time.Duration //Время действия jwt для авторизации
}

func Create() (*Config, error) {
	cfg := &Config{}

	var JWTKey, JWTDuration string

	flag.StringVar(&cfg.Endpoint, "a", "localhost:8081", "address and port to run server")
	flag.StringVar(&cfg.DataBaseDNS, "d", "", "db dns")
	flag.StringVar(&cfg.AccrualAddress, "r", "localhost:8080", "accrual address")
	flag.StringVar(&JWTKey, "k", "aL6HmkWp7D", "JWT key")
	flag.StringVar(&JWTDuration, "t", "60m", "JWT duration")

	flag.Parse()

	/*Но если заданы в окружении - берем оттуда*/
	if addr, exist := os.LookupEnv("RUN_ADDRESS"); exist {
		cfg.Endpoint = addr
	}

	if dns, exist := os.LookupEnv("DATABASE_URI"); exist {
		logger.Info("DATABASE_URI env: %s", dns)
		cfg.DataBaseDNS = dns
	}

	if cfg.DataBaseDNS == `` {
		return nil, fmt.Errorf("db dns is empty")
	}

	if accrual, exist := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); exist {
		logger.Info("ACCRUAL_SYSTEM_ADDRESS env: %s", accrual)
		cfg.AccrualAddress = accrual
	}

	if cfg.AccrualAddress == `` {
		return nil, fmt.Errorf("accrual address is empty")
	}

	if key, exist := os.LookupEnv("JWTKey"); exist {
		JWTKey = key
	}

	if duration, exist := os.LookupEnv("JWTDuration"); exist {
		JWTDuration = duration
	}

	cfg.JWTKey = []byte(JWTKey)
	if duration, err := time.ParseDuration(JWTDuration); err != nil {
		return nil, fmt.Errorf("JWT DURATION: %w", err)
	} else {
		cfg.JWTDuration = duration
	}

	return cfg, nil
}
