package sandbox

// ResourceLimits 资源限制配置
type ResourceLimits struct {
	// CPU 限制
	CPUQuota  int64 // CPU 配额（微秒/周期）
	CPUPeriod int64 // CPU 周期（微秒）

	// 内存限制
	MemoryLimit string // 内存限制（如 "512m", "4g"）
	MemorySwap  string // swap 限制

	// 进程限制
	PIDsLimit int64 // 最大进程数

	// 磁盘限制
	DiskQuota string // 磁盘配额
	DiskRate  string // IO 速率限制

	// 网络限制
	NetRate string // 网络速率限制
}

// DefaultLimits 默认安全限制
func DefaultLimits() ResourceLimits {
	return ResourceLimits{
		CPUQuota:    50000, // 50% of 1 CPU
		CPUPeriod:   100000,
		MemoryLimit: "1g",
		MemorySwap:  "1g", // 禁用 swap
		PIDsLimit:   100,
		DiskQuota:   "10g",
		DiskRate:    "10mb",
		NetRate:     "100mbps",
	}
}

// AttackerLimits 攻击者容器限制
func AttackerLimits() ResourceLimits {
	return ResourceLimits{
		CPUQuota:    200000, // 2 CPUs
		CPUPeriod:   100000,
		MemoryLimit: "4g",
		MemorySwap:  "4g",
		PIDsLimit:   200,
		DiskQuota:   "20g",
		DiskRate:    "50mb",
		NetRate:     "100mbps",
	}
}

// TargetLimits 靶机容器限制
func TargetLimits() ResourceLimits {
	return ResourceLimits{
		CPUQuota:    100000, // 1 CPU
		CPUPeriod:   100000,
		MemoryLimit: "2g",
		MemorySwap:  "2g",
		PIDsLimit:   50,
		DiskQuota:   "5g",
		DiskRate:    "10mb",
		NetRate:     "50mbps",
	}
}
