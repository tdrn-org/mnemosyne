/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"log/slog"

	"github.com/tdrn-org/go-config-toml"
	"github.com/tdrn-org/go-log"
)

type LoggingConfig struct {
	Level          LogLevel  `toml:"level"`
	Target         LogTarget `toml:"target"`
	Color          LogColor  `toml:"color"`
	FileName       string    `toml:"file_name"`
	FileSizeLimit  int64     `toml:"file_size_limit"`
	SyslogNetwork  string    `toml:"syslog_network"`
	SyslogAddress  string    `toml:"syslog_address"`
	SyslogEncoding string    `toml:"syslog_encoding"`
	SyslogFacility int       `toml:"syslog_facility"`
}

type LogLevel slog.Level

var logLevelMarshalMap map[LogLevel]string = map[LogLevel]string{
	LogLevel(slog.LevelDebug): "debug",
	LogLevel(slog.LevelInfo):  "info",
	LogLevel(slog.LevelWarn):  "warn",
	LogLevel(slog.LevelError): "error",
}

var logLevelUnmarshalMap map[string]LogLevel = map[string]LogLevel{
	"debug": LogLevel(slog.LevelDebug),
	"info":  LogLevel(slog.LevelInfo),
	"warn":  LogLevel(slog.LevelWarn),
	"error": LogLevel(slog.LevelError),
}

func (l LogLevel) String() string {
	return logLevelMarshalMap[l]
}

func (l LogLevel) MarshalText() ([]byte, error) {
	return config.MarshalEnum(l, logLevelMarshalMap)
}

func (l *LogLevel) UnmarshalText(text []byte) error {
	logLevel, err := config.UnmarshalEnum(logLevelUnmarshalMap, text)
	if err != nil {
		return err
	}
	*l = logLevel
	return nil
}

type LogTarget log.Target

var logTargetMarshalMap map[LogTarget]string = map[LogTarget]string{
	LogTarget(log.TargetStdout):     string(log.TargetStdout),
	LogTarget(log.TargetStdoutText): string(log.TargetStdoutText),
	LogTarget(log.TargetStdoutJSON): string(log.TargetStdoutJSON),
	LogTarget(log.TargetStderr):     string(log.TargetStderr),
	LogTarget(log.TargetStderrText): string(log.TargetStderrText),
	LogTarget(log.TargetStderrJSON): string(log.TargetStderrJSON),
	LogTarget(log.TargetFileText):   string(log.TargetFileText),
	LogTarget(log.TargetFileJSON):   string(log.TargetFileJSON),
	LogTarget(log.TargetSyslog):     string(log.TargetSyslog),
}

var logTargetUnmarshalMap map[string]LogTarget = map[string]LogTarget{
	string(log.TargetStdout):     LogTarget(log.TargetStdout),
	string(log.TargetStdoutText): LogTarget(log.TargetStdoutText),
	string(log.TargetStdoutJSON): LogTarget(log.TargetStdoutJSON),
	string(log.TargetStderr):     LogTarget(log.TargetStderr),
	string(log.TargetStderrText): LogTarget(log.TargetStderrText),
	string(log.TargetStderrJSON): LogTarget(log.TargetStderrJSON),
	string(log.TargetFileText):   LogTarget(log.TargetFileText),
	string(log.TargetFileJSON):   LogTarget(log.TargetFileJSON),
	string(log.TargetSyslog):     LogTarget(log.TargetSyslog),
}

func (t LogTarget) String() string {
	return logTargetMarshalMap[t]
}

func (t LogTarget) MarshalText() ([]byte, error) {
	return config.MarshalEnum(t, logTargetMarshalMap)
}

func (t *LogTarget) UnmarshalText(text []byte) error {
	logTarget, err := config.UnmarshalEnum(logTargetUnmarshalMap, text)
	if err != nil {
		return err
	}
	*t = logTarget
	return nil
}

type LogColor log.Color

var logColorMarshalMap map[LogColor]string = map[LogColor]string{
	LogColor(log.ColorAuto): "auto",
	LogColor(log.ColorOff):  "off",
	LogColor(log.ColorOn):   "on",
}

var logColorUnmarshalMap map[string]LogColor = map[string]LogColor{
	"auto": LogColor(log.ColorAuto),
	"off":  LogColor(log.ColorOff),
	"on":   LogColor(log.ColorOn),
}

func (c LogColor) String() string {
	return logColorMarshalMap[c]
}

func (c LogColor) MarshalText() ([]byte, error) {
	return config.MarshalEnum(c, logColorMarshalMap)
}

func (c *LogColor) UnmarshalText(text []byte) error {
	logColor, err := config.UnmarshalEnum(logColorUnmarshalMap, text)
	if err != nil {
		return err
	}
	*c = logColor
	return nil
}
