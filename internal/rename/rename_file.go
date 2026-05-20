package rename

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
)

// RenameFile 将文件重命名为目标路径；目标已存在时返回错误。
// 跨文件系统时自动回退为复制后删除源文件。
func RenameFile(fromPath, toPath string) error {
	fromPath = strings.TrimSpace(fromPath)
	toPath = strings.TrimSpace(toPath)
	if fromPath == "" || toPath == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	if fromPath == toPath {
		return nil
	}
	if _, err := os.Stat(fromPath); err != nil {
		return fmt.Errorf("源文件不存在: %w", err)
	}
	if _, err := os.Stat(toPath); err == nil {
		return fmt.Errorf("目标文件已存在: %s", toPath)
	}
	_ = ensureRenamePermissions(fromPath, toPath)
	if err := renameWithFallback(fromPath, toPath); err != nil {
		return err
	}
	return nil
}

func renameWithFallback(fromPath, toPath string) error {
	err := os.Rename(fromPath, toPath)
	if err == nil {
		return nil
	}
	if shouldRenameByCopy(err) || isPermissionError(err) {
		_ = ensureRenamePermissions(fromPath, toPath)
		if err2 := os.Rename(fromPath, toPath); err2 == nil {
			return nil
		}
		if copyErr := renameByCopy(fromPath, toPath); copyErr == nil {
			return nil
		}
		if isPermissionError(err) {
			return fmt.Errorf("重命名文件失败: %w（%s）", err, PermissionHint(fromPath, toPath))
		}
	}
	if isPermissionError(err) {
		return fmt.Errorf("重命名文件失败: %w（%s）", err, PermissionHint(fromPath, toPath))
	}
	return fmt.Errorf("重命名文件失败: %w", err)
}

func shouldRenameByCopy(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}

func isPermissionError(err error) bool {
	return errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)
}

func renameByCopy(fromPath, toPath string) error {
	if err := copyFile(fromPath, toPath); err != nil {
		return err
	}
	if err := os.Remove(fromPath); err != nil {
		_ = os.Remove(toPath)
		return fmt.Errorf("删除源文件失败: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	if info, err := os.Stat(src); err == nil {
		_ = os.Chmod(dst, info.Mode())
	}
	return nil
}
