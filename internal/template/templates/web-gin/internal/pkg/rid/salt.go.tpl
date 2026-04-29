package rid

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"os"
)

// Salt 计算 machine ID 的 hash 值作为短码生成的盐值（uint64）。
func Salt() uint64 {
	// 使用 FNV-1a 哈希算法计算 machine ID 的 hash 值。
	hasher := fnv.New64a()
	hasher.Write(ReadMachineID())

	return hasher.Sum64()
}

// ReadMachineID 尝试读取本机 machine ID；失败时退化为生成随机 ID。
func ReadMachineID() []byte {
	id := make([]byte, 3)
	machineID, err := readPlatformMachineID()
	if err != nil || len(machineID) == 0 {
		machineID, err = os.Hostname()
	}

	if err == nil && len(machineID) != 0 {
		hasher := sha256.New()
		hasher.Write([]byte(machineID))
		copy(id, hasher.Sum(nil))
	} else {
		// machine ID 与 hostname 都拿不到时，回退到随机数。
		if _, randErr := rand.Reader.Read(id); randErr != nil {
			panic(fmt.Errorf("id: cannot get hostname nor generate a random number: %w; %w", err, randErr))
		}
	}
	return id
}

// readPlatformMachineID 尝试从平台特定路径读取 machine ID。
func readPlatformMachineID() (string, error) {
	data, err := os.ReadFile("/etc/machine-id")
	if err != nil || len(data) == 0 {
		data, err = os.ReadFile("/sys/class/dmi/id/product_uuid")
	}
	return string(data), err
}
