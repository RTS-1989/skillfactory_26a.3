package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
)

var (
	infoLog  = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
)

type RingBuffer struct {
	values []int
	pos    int
	size   int
	m      sync.RWMutex
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		values: make([]int, size),
		pos:    -1,
		size:   size,
		m:      sync.RWMutex{},
	}
}

func (r *RingBuffer) Push(elementToInsert int) {
	infoLog.Println("Push element to buffer")
	r.m.Lock()
	defer r.m.Unlock()
	if r.pos == r.size-1 {
		for i := 1; i <= r.size-1; i++ {
			r.values[i-1] = r.values[i]
		}
		r.values[r.pos] = elementToInsert
	} else {
		r.pos++
		r.values[r.pos] = elementToInsert
	}
}

func (r *RingBuffer) Get() []int {
	infoLog.Println("Get element to buffer")
	valuesToGet := make([]int, 0, r.size)
	for _, value := range r.values {
		if value != 0 {
			valuesToGet = append(valuesToGet, value)
		}
	}
	r.values = make([]int, r.size)
	return valuesToGet
}

type Stage func(<-chan int, <-chan bool, *sync.WaitGroup) chan int

type Pipeline struct {
	stagesList []Stage
	done       chan bool
	wg         *sync.WaitGroup
}

func newPipeline(done chan bool, wg *sync.WaitGroup, stages ...Stage) *Pipeline {
	return &Pipeline{
		done:       done,
		wg:         wg,
		stagesList: stages,
	}
}

func (pl *Pipeline) runStage(stage Stage, inChan <-chan int, done chan bool) chan int {
	return stage(inChan, done, pl.wg)
}

func (pl *Pipeline) RunPipeline(producerChan <-chan int) <-chan int {
	var pipelineChannel = producerChan
	for n, stage := range pl.stagesList {
		infoLog.Printf("Start run stage № %d\n", n+1)
		pl.wg.Add(1)
		pipelineChannel = pl.runStage(stage, pipelineChannel, pl.done)
	}
	return pipelineChannel
}

// filterNegative method for filtering negative values. Returns filtered channel of integers
func filterNegative(producerChannel <-chan int, done <-chan bool, wg *sync.WaitGroup) chan int {
	infoLog.Println("Run stage filterNegative")
	negativeValuesCounter := 0
	filteredNegative := make(chan int)

	go func() {
		defer close(filteredNegative)
		for {
			select {
			case value := <-producerChannel:
				if value >= 0 {
					filteredNegative <- value
					negativeValuesCounter += 1
				}
			case <-done:
				if len(producerChannel) > 0 {
					for value := range producerChannel {
						if value >= 0 {
							filteredNegative <- value
							negativeValuesCounter += 1
						}
					}
				}
				infoLog.Printf("Pipeline get %d not negative values\n", negativeValuesCounter)
				infoLog.Println("Останавливаем канал фильтрации негативных значений")
				wg.Done()
				return
			}
		}
	}()

	return filteredNegative
}

// filterMultipleOfThree method. Filters multiply by 3 values. Returns filtered channel of integers
func filterMultipleOfThree(negativesChan <-chan int, done <-chan bool, wg *sync.WaitGroup) chan int {
	infoLog.Println("Run stage filterMultipleOfThree")
	multipleOfThreeValuesCounter := 0
	multipleThree := make(chan int)

	go func() {
		defer close(multipleThree)
		for {
			select {
			case value := <-negativesChan:
				if value%3 == 0 && value != 0 {
					multipleThree <- value
					multipleOfThreeValuesCounter += 1
				}
			case <-done:
				if len(negativesChan) > 0 {
					for value := range negativesChan {
						if value%3 == 0 && value != 0 {
							multipleThree <- value
							multipleOfThreeValuesCounter += 1
						}
					}
				}
				infoLog.Printf("Pipeline get %d not multiple by 3 values\n", multipleOfThreeValuesCounter)
				infoLog.Println("Останавливаем канал фильтрации значений не кратных 3")
				wg.Done()
				return
			}
		}
	}()

	return multipleThree
}

type Producer struct {
	doneCh            chan bool
	valuesSentCounter int
	wg                *sync.WaitGroup
}

func newProducer(done chan bool, wg *sync.WaitGroup) *Producer {
	return &Producer{
		doneCh:            done,
		valuesSentCounter: 0,
		wg:                wg,
	}
}

// startProducer method of Producer. Sends value from command line to channel of integers. Returns channel of integers
func (p *Producer) startProducer() chan int {
	infoLog.Println("Start producer")
	var value int
	channel := make(chan int)
	ticker := time.NewTicker(50 * time.Millisecond)
	go func(c chan int) {
		// не хватает return
		p.wg.Add(1)
		defer close(c)
		for {
			select {
			case <-ticker.C:
				_, err := fmt.Scanf("%d", &value)
				if err != nil {
					errorLog.Println("Value is not int!")
				} else {
					c <- value
					p.valuesSentCounter += 1
					infoLog.Println("Producer send value")
				}
			}
		}
	}(channel)

	go func() {
		select {
		case <-p.doneCh:
			infoLog.Println("Stop producer")
			p.wg.Done()
			return
		}
	}()

	return channel
}

type Consumer struct {
	doneCh            chan bool
	valuesConsumerGet int
	wg                *sync.WaitGroup
}

//newConsumer returns Consumer instance
func newConsumer(done chan bool, wg *sync.WaitGroup) *Consumer {
	return &Consumer{
		doneCh:            done,
		valuesConsumerGet: 0,
		wg:                wg,
	}
}

func (cn *Consumer) startConsumer(filteredByThree <-chan int) chan int {
	infoLog.Println("Start consumer")
	channel := make(chan int)
	go func(filteredValues chan int) {
		cn.wg.Add(1)
		defer close(filteredValues)
		for {
			select {
			case value := <-filteredByThree:
				filteredValues <- value
				cn.valuesConsumerGet += 1
				infoLog.Println("Consumer get value")
			case <-cn.doneCh:
				if len(filteredByThree) > 0 {
					for value := range filteredByThree {
						filteredValues <- value
					}
				}
				infoLog.Println("Stop consumer")
				cn.wg.Done()
				return
			}
		}
	}(channel)

	return channel
}

func main() {
	infoLog.Printf("Start app\n")
	const ringBufferSize = 10
	buffer := NewRingBuffer(ringBufferSize)
	bufferTicker := time.NewTicker(10 * time.Second)
	doneCh := make(chan bool)

	producerWg := sync.WaitGroup{}
	consumerWg := sync.WaitGroup{}
	stagesWg := sync.WaitGroup{}

	producer := newProducer(doneCh, &producerWg)
	consumer := newConsumer(doneCh, &consumerWg)
	pipeline := newPipeline(doneCh, &stagesWg, filterNegative, filterMultipleOfThree)

	stopAppChan := make(chan os.Signal, 1)
	signal.Notify(stopAppChan, os.Interrupt)

	producerChan := producer.startProducer()
	pipelineChannel := pipeline.RunPipeline(producerChan)
	consumerChannel := consumer.startConsumer(pipelineChannel)

	infoLog.Printf("Please insert values in buffer\n")

	for {
		select {
		case value := <-consumerChannel:
			buffer.Push(value)
			infoLog.Printf("Current buffer values -> %v\n", buffer.values)
		case <-bufferTicker.C:
			infoLog.Printf("Values got from buffer -> %v\n", buffer.Get())
		case <-stopAppChan:
			infoLog.Printf("\nStatistics:\n")
			infoLog.Printf("Producer sent %d values\n", producer.valuesSentCounter)
			infoLog.Printf("Consumer get %d values\n", consumer.valuesConsumerGet)
			close(doneCh)
			infoLog.Printf("Values got from buffer -> %v\n", buffer.Get())
			stagesWg.Wait()
			consumerWg.Wait()
			producerWg.Wait()
			infoLog.Println("App stopped")
			return
		}
	}
}
