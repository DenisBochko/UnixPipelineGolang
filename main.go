package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	Md5_limit int8 = 1
)

func main() {
	timePoint1 := time.Now()

	workers := []job{
		func(in, out chan interface{}) {
			defer close(out)
			for _, v := range []string{"0", "1", "2", "3", "4"} {
				out <- v
			}
		},
		SingleHash,
		MultiHash,
		CombineResults,
	}

	ExecutePipeline(workers...)

	timePoint2 := time.Now()
	fmt.Printf("Прогорамма отработала за %v", timePoint2.Sub(timePoint1))
}

func ExecutePipeline(workers ...job) {
	// Создаём слайс каналов
	Channels := make([]chan interface{}, 0, 10)
	// Cоздаём нужное кол-во каналов и кладём в слайс
	for i := 0; i <= len(workers)+1; i++ {
		Channels = append(Channels, make(chan interface{}, 100))
	}
	// Создаём WairGroup для проверки: отработали ли все воркеры, переданные в функцию
	var wg sync.WaitGroup

	wg.Add(len(workers))

	for i := 0; i < len(workers); i++ {
		// запускаем воркеры в отдельных горутинах, передающие данные между собой через каналы
		go func(i int) {
			defer wg.Done()
			workers[i](Channels[i], Channels[i+1])
		}(i)
	}

	wg.Wait()
}

var SingleHash = func(inChan, outChan chan interface{}) {
	defer close(outChan)

	wg := sync.WaitGroup{}
	quotaCh := make(chan struct{}, Md5_limit) // создаём канал квоты, который не позволяет работать больше 1 функции DataSignerMd5 в один момент времени

	for data := range inChan {
		data := data.(string) // показываем программе, что объект типа interface{} это именно строка

		wg.Add(1)
		go func(value string) {
			defer wg.Done()

			wgLocal := sync.WaitGroup{}
			FirstCrc32 := ""
			Md5 := ""
			SecondCrc32 := ""

			wgLocal.Add(1)
			go func() {
				defer wgLocal.Done()
				FirstCrc32 = DataSignerCrc32(value)
			}()

			wgLocal.Add(1)
			go func() {
				defer wgLocal.Done()
				quotaCh <- struct{}{} // захватили управление
				Md5 = DataSignerMd5(value)
				<-quotaCh                          // вернули управление
				SecondCrc32 = DataSignerCrc32(Md5) // досчитали
			}()

			wgLocal.Wait()
			outChan <- FirstCrc32 + "~" + SecondCrc32
		}(data)

	}
	wg.Wait()
}

var MultiHash = func(inChan, outChan chan interface{}) {
	defer close(outChan)

	wg := sync.WaitGroup{}

	for data := range inChan {
		data := data.(string) // показываем программе, что объект типа interface{} это именно строка

		wg.Add(1)
		go func(value string) {
			defer wg.Done()

			result := ""
			wgLocal := sync.WaitGroup{}

			handler := func(variable *string, th, previous string, wg *sync.WaitGroup) {
				defer wg.Done()
				res := DataSignerCrc32(th + previous)
				*variable = res
			}

			Ch1, Ch2, Ch3, Ch4, Ch5, Ch6 := "", "", "", "", "", ""

			wgLocal.Add(6)
			go handler(&Ch1, "0", value, &wgLocal)
			go handler(&Ch2, "1", value, &wgLocal)
			go handler(&Ch3, "2", value, &wgLocal)
			go handler(&Ch4, "3", value, &wgLocal)
			go handler(&Ch5, "4", value, &wgLocal)
			go handler(&Ch6, "5", value, &wgLocal)
			wgLocal.Wait()

			result = Ch1 + Ch2 + Ch3 + Ch4 + Ch5 + Ch6
			outChan <- result
		}(data)
	}
	wg.Wait()
}

var CombineResults = func(inChan, outChan chan interface{}) {
	defer close(outChan)
	result := make([]string, 0, 100)
	for data := range inChan {
		outChan <- data
		data := data.(string) // показываем программе, что объект типа interface{} это именно строка
		result = append(result, data)
	}
	sort.Strings(result)
	fmt.Println(strings.Join(result, "_"))
}

// для входных []string{"0", "1"}
// 29568666068035183841425683795340791879727309630931025356555_4958044192186797981418233587017209679042592862002427381542
