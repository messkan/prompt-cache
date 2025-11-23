package semantic

import (
	"encoding/binary"
	"math"
)

func CosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float32

	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

func BytesToFloat32(data []byte) []float32 {
	size := len(data) / 4
	res := make([]float32, size)
	for i := 0; i < size; i++ {
		bits := binary.LittleEndian.Uint32(data[i*4 : i*4+4])
		res[i] = math.Float32frombits(bits)
	}
	return res
}

func Float32ToBytes(data []float32) []byte {
	size := len(data) * 4
	res := make([]byte, size)
	for i, v := range data {
		bits := math.Float32bits(v)
		binary.LittleEndian.PutUint32(res[i*4:], bits)
	}
	return res
}
