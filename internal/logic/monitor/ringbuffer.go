// =================================================================================
// Ring Buffer - 环形缓冲区
// =================================================================================

package monitor

import (
	"sync"
)

// sRingBuffer 环形缓冲区（线程安全）
type sRingBuffer[T any] struct {
	data  []T
	size  int
	head  int
	tail  int
	count int
	mu    sync.RWMutex
}

// NewRingBuffer 创建环形缓冲区
func NewRingBuffer[T any](size int) *sRingBuffer[T] {
	return &sRingBuffer[T]{
		data: make([]T, size),
		size: size,
	}
}

// Put 添加元素
func (rb *sRingBuffer[T]) Put(item T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.data[rb.tail] = item
	rb.tail = (rb.tail + 1) % rb.size

	if rb.count < rb.size {
		rb.count++
	} else {
		// 缓冲区已满，移动head
		rb.head = (rb.head + 1) % rb.size
	}
}

// GetAll 获取所有元素
func (rb *sRingBuffer[T]) GetAll() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return nil
	}

	result := make([]T, rb.count)
	for i := 0; i < rb.count; i++ {
		idx := (rb.head + i) % rb.size
		result[i] = rb.data[idx]
	}
	return result
}

// GetLast 获取最后N个元素
func (rb *sRingBuffer[T]) GetLast(n int) []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return nil
	}

	if n > rb.count {
		n = rb.count
	}

	result := make([]T, n)
	start := rb.count - n
	for i := 0; i < n; i++ {
		idx := (rb.head + start + i) % rb.size
		result[i] = rb.data[idx]
	}
	return result
}

// Count 获取元素数量
func (rb *sRingBuffer[T]) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Capacity 获取缓冲区容量
func (rb *sRingBuffer[T]) Capacity() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// Clear 清空缓冲区
func (rb *sRingBuffer[T]) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.tail = 0
	rb.count = 0
	rb.data = make([]T, rb.size)
}
