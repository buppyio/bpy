package bpy

type CStore interface {
	Put([]byte) ([32]byte, error)
	Get([32]byte) ([]byte, error)
}
