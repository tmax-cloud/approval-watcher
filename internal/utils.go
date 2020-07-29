package internal

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

func Users(cmPath string) (map[string]string, error) {
	file, err := os.Open(cmPath)
	if err != nil {
		return nil, errors.New("could not open config map")
	}
	defer file.Close()

	users := make(map[string]string)
	scanner := bufio.NewScanner(file)
	// Parse line-separated
	for scanner.Scan() {
		// Parse comma-separated
		userList := strings.Split(scanner.Text(), ",")
		for i := range userList {
			userList[i] = strings.TrimSpace(userList[i])

			user := strings.Split(userList[i], "=")
			users[user[0]] = user[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.New("error occurs on scanner")
	}

	return users, nil
}

func GenerateUserLabel(list []string) map[string]string {
	result := map[string]string{}

	for _, user := range list {
		result[user] = ""
	}

	return result
}
