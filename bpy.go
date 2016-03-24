package bpy

type CStoreReader interface {
	Get([32]byte) ([]byte, error)
	Close() error
}

type CStoreWriter interface {
	Put([]byte) ([32]byte, error)
	Close() error
}
