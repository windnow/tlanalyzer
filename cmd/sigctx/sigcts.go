package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func worker(ctx context.Context, name string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("%s: Работа завершена\n", name)
			return
		default:
			fmt.Printf("%s: Выполняется работа\n", name)
			time.Sleep(1 * time.Second)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go worker(ctx, "Горутина 1", wg)
	go worker(ctx, "Горутина 2", wg)
	go worker(ctx, "Горутина 3", wg)

	// Создаем канал для получения сигналов ОС
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Ожидаем сигналы ОС
		sig := <-sigCh
		fmt.Println("Получен сигнал:", sig)
		cancel() // Отменяем контекст
	}()

	wg.Wait()

	fmt.Println("Программа завершена")
}
