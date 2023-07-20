package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type compressWriter struct {
	once sync.Once

	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w: w,
	}
}

func (m *compressWriter) Header() http.Header {
	return m.w.Header()
}

func (m *compressWriter) Write(p []byte) (int, error) {
	//Перед началом отгрузки данных надо проверить тип данных, и если можно сжимать - ставим заголовок и создаем
	m.once.Do(func() {
		for _, v := range m.w.Header()["Content-Type"] {
			if v == "application/json" || v == "text/html" {
				m.w.Header().Set("Content-Encoding", "gzip")
				m.zw = gzip.NewWriter(m.w)
				break
			}
		}
	})

	if m.zw != nil {
		return m.zw.Write(p)
	} else {
		return m.w.Write(p)
	}
}

func (m *compressWriter) WriteHeader(statusCode int) {
	m.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (m *compressWriter) Close() error {
	if m.zw != nil {
		return m.zw.Close()
	} else {
		return nil
	}
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ow := w
		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw := newCompressWriter(w)
			// меняем оригинальный http.ResponseWriter на новый
			ow = cw
			// не забываем отправить клиенту все сжатые данные после завершения middleware
			defer cw.Close()
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// меняем тело запроса на новое
			r.Body = cr
			defer cr.Close()
		}

		// передаём управление хендлеру
		h.ServeHTTP(ow, r)
	})
}
