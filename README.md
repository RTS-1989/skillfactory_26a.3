Принцип работы

Данное приложение представляет собой реализацию паттерна "pipeline".

При старте приложения создаются "producer", "pipeline" и "consumer".

"Producer" при запуске метода "startProducer" обрабатывает данные из консоли и возвращает
канал, которые передается "pipeline".

При запуске метода "RunPipeline", этому методу передается канал созданный "producer".
Далее в "pipeline" обрабатываются, созданные для объекта "pipeline" стадии. В итоге
"RunPipeline" возвращает канал, который передается "consumer".

При старте метода "startConsumer" данные из канала отфильтрованного в "pipeline" 
передаются "consumer", которые записывает данные в свой канал. 

"startConsumer" возвращает свой канал, при записи данных в который начинают отправляться в 
кольцевой буфер. Данные в кольцевом буфере хранятся ограниченное время после чего
буфер очищается. 

Для остановки приложения нужно нажать сочетание клавиш "CTRL-C".