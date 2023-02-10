package kyber_wrap

import "io"

func (s *Scalar) String() string {
	return s.fr.GetString(10)
}

func (s *Scalar) MarshalSize() int {
	return s.fr.Len()
}

func (s *Scalar) MarshalTo(w io.Writer) (int, error) {
	b, _ := s.MarshalBinary()
	return w.Write(b)
}

func (s *Scalar) UnmarshalFrom(r io.Reader) (int, error) {
	b := make([]byte, s.fr.Len())
	n, _ := r.Read(b)
	return n, s.fr.InterpretFrom(b)
}

func (s *Scalar) MarshalBinary() ([]byte, error) {
	b := make([]byte, s.fr.Len())
	s.fr.PackTo(b)
	return b, nil
}

func (s *Scalar) UnmarshalBinary(data []byte) error {
	return s.fr.InterpretFrom(data)
}

func (p *Point) String() string {
	return p.g2.GetString(10)
}

func (p *Point) MarshalSize() int {
	return p.g2.Len()
}

func (p *Point) MarshalTo(w io.Writer) (int, error) {
	b, _ := p.MarshalBinary()
	return w.Write(b)
}

func (p *Point) UnmarshalFrom(r io.Reader) (int, error) {
	b := make([]byte, p.g2.Len())
	n, _ := r.Read(b)
	return n, p.g2.InterpretFrom(b)
}

func (p *Point) MarshalBinary() ([]byte, error) {
	b := make([]byte, p.g2.Len())
	p.g2.PackTo(b)
	return b, nil
}

func (p *Point) UnmarshalBinary(data []byte) error {
	return p.g2.InterpretFrom(data)
}
