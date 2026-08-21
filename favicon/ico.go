//go:build !wasm

package favicon

import "encoding/binary"

// encodeICO crea un .ico de una sola imagen a partir del PNG ya generado.
// Cabecera de 22 bytes en little-endian + bytes del PNG de 32x32.
func encodeICO(pngData []byte) []byte {
	hdr := make([]byte, 22)
	// ICONDIR
	binary.LittleEndian.PutUint16(hdr[0:2], 0) // reservado
	binary.LittleEndian.PutUint16(hdr[2:4], 1) // tipo icono
	binary.LittleEndian.PutUint16(hdr[4:6], 1) // cantidad
	// ICONDIRENTRY
	hdr[6] = 32                                                     // ancho
	hdr[7] = 32                                                     // alto
	hdr[8] = 0                                                      // colores
	hdr[9] = 0                                                      // reservado
	binary.LittleEndian.PutUint16(hdr[10:12], 1)                    // planos
	binary.LittleEndian.PutUint16(hdr[12:14], 32)                   // bpp
	binary.LittleEndian.PutUint32(hdr[14:18], uint32(len(pngData))) // tamaño
	binary.LittleEndian.PutUint32(hdr[18:22], 22)                   // offset

	out := make([]byte, 0, len(hdr)+len(pngData))
	out = append(out, hdr...)
	out = append(out, pngData...)
	return out
}
