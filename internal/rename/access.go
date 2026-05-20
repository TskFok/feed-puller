package rename

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// PermissionHint 在权限类错误时附加进程身份与路径可访问性说明。
func PermissionHint(fromPath, toPath string) string {
	euid, egid := os.Geteuid(), os.Getegid()
	return fmt.Sprintf(
		"进程 uid=%d gid=%d；%s；目标父目录 %s；%s",
		euid,
		egid,
		describePath(fromPath, euid),
		describeDir(filepath.Dir(toPath), euid),
		renamePermissionAdvice(fromPath, euid),
	)
}

func renamePermissionAdvice(fromPath string, euid int) string {
	info, err := os.Stat(fromPath)
	if err != nil {
		return ""
	}
	fuid, _ := fileIDs(info)
	dirInfo, err := os.Stat(filepath.Dir(fromPath))
	if err != nil {
		return ""
	}
	duid, _ := fileIDs(dirInfo)
	if euid != 0 && fuid >= 0 && euid != fuid {
		return fmt.Sprintf("提示：文件属主 uid=%d 与进程 uid=%d 不一致，请将 PUID/PGID 设为 aria2 写文件的用户（不要用 0）", fuid, euid)
	}
	if !dirIsWritable(filepath.Dir(fromPath)) && duid >= 0 && euid != duid && euid != 0 {
		return fmt.Sprintf("提示：重命名需要目录写权限，目录属主 uid=%d，进程 uid=%d", duid, euid)
	}
	return "提示：重命名需要父目录写权限，请确认 PUID/PGID 与 aria2 一致"
}

func describePath(path string, euid int) string {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Sprintf("源路径 %q stat 失败: %v", path, err)
	}
	uid, gid := fileIDs(info)
	return fmt.Sprintf(
		"源路径 %q mode=%04o uid=%d gid=%d%s%s",
		path,
		info.Mode().Perm(),
		uid,
		gid,
		ownershipHint(uid, euid),
		dirWriteHint(filepath.Dir(path)),
	)
}

func describeDir(dir string, euid int) string {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Sprintf("%q stat 失败: %v", dir, err)
	}
	uid, gid := fileIDs(info)
	return fmt.Sprintf("%q mode=%04o uid=%d gid=%d%s%s", dir, info.Mode().Perm(), uid, gid, ownershipHint(uid, euid), dirWriteHint(dir))
}

func ownershipHint(fileUID, euid int) string {
	if fileUID < 0 {
		return ""
	}
	if fileUID == euid {
		return "（属主与进程一致）"
	}
	return fmt.Sprintf("（属主 uid=%d ≠ 进程 uid=%d）", fileUID, euid)
}

func dirWriteHint(dir string) string {
	if dirIsWritable(dir) {
		return "（目录可写）"
	}
	return "（目录不可写）"
}

func dirIsWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".feed-puller-write-check-*")
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	return true
}

// ensureRenamePermissions 在重命名前尽量保证父目录可写（仅当进程为 root 或目录属主时尝试 chmod）。
func ensureRenamePermissions(fromPath, toPath string) error {
	for _, dir := range []string{filepath.Dir(fromPath), filepath.Dir(toPath)} {
		if dirIsWritable(dir) {
			continue
		}
		if err := tryChmodDirWritable(dir); err != nil {
			return err
		}
	}
	return nil
}

func tryChmodDirWritable(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	duid, _ := fileIDs(info)
	euid := os.Geteuid()
	if euid != 0 && euid != duid {
		return fmt.Errorf("目录 %q 不可写且进程 uid=%d 不是目录属主 uid=%d", dir, euid, duid)
	}
	mode := info.Mode().Perm() | 0200
	if euid == 0 {
		mode = info.Mode().Perm() | 0220
		if mode&0111 == 0 {
			mode |= 0111
		}
	}
	if err := os.Chmod(dir, mode); err != nil {
		return fmt.Errorf("无法调整目录 %q 权限: %w", dir, err)
	}
	if !dirIsWritable(dir) {
		return fmt.Errorf("目录 %q 仍不可写", dir)
	}
	return nil
}

func fileIDs(info os.FileInfo) (uid, gid int) {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return int(st.Uid), int(st.Gid)
	}
	return -1, -1
}
