/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package consensus

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	flags := MergeCmd.Flags()

	flags.String("filea", "", "first file with merge")
	flags.String("fileb", "", "second file with merge")
	flags.String("filec", "", "merge dest file ")

}

var MergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "create genesis address file",
	RunE:  merge,
}

func merge(cmd *cobra.Command, args []string) error {
	MergeFile(viper.GetString("filea"), viper.GetString("fileb"), viper.GetString("filec"))
	return nil
}

// filea+fileb->filec
func MergeFile(filea, fileb, filec string) error {
	file, err := os.OpenFile(filec, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0664)
	if err != nil {
		fmt.Printf("open filec err=%v\n", err)
		return err
	}
	defer file.Close()

	data, err := ioutil.ReadFile(filea)
	if err != nil {
		fmt.Printf("read filea  err=%v\n", err)
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		fmt.Printf("write into filec err=%v\n", err)
		return err
	}
	data, err = ioutil.ReadFile(fileb)
	if err != nil {
		fmt.Printf("read fileb  err=%v\n", err)
		return err
	}
	_, err = file.Write(data)
	if err != nil {
		fmt.Printf("write into filec err=%v\n", err)
		return err
	}
	file.Sync()
	return nil
}
func AppendString(filename, content string) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0664)
	if err != nil {
		fmt.Printf("err=%v\n", err)
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content + "\n")
	if err != nil {
		fmt.Printf("err=%v\n", err)
		return err
	}
	file.Sync()

	return nil
}
func WriteStringStart(templetefilename, filename, content string) error {
	file, err := os.Open(templetefilename)
	if err != nil {
		fmt.Printf("err=%v\n", err)
		return err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("err=%v\n", err)
		return err
	}
	writeData := []byte(content)
	writeData = append(writeData, data...)

	err = ioutil.WriteFile(filename, writeData, 0666)
	if err != nil {
		fmt.Printf("err=%v\n", err)
		return err
	}
	return nil
}
