package pswarm

import (
	"math"
	"math/rand"

	"github.com/rwcarlsen/optim"
	"github.com/rwcarlsen/optim/mesh"
)

type Particle struct {
	Id int
	optim.Point
	Vel  []float64
	Best optim.Point
}

func (p *Particle) Update(newp optim.Point) {
	p.Val = newp.Val
	if p.Val < p.Best.Val || p.Best.Len() == 0 {
		p.Best = newp
	}
}

type Population []*Particle

// NewPopulation initializes a population of particles using the given points
// and generates velocities for each dimension i initialized to uniform random
// values between minv[i] and maxv[i].  github.com/rwcarlsen/optim.Rand is
// used for random numbers.
func NewPopulation(points []optim.Point, minv, maxv []float64) Population {
	pop := make(Population, len(points))
	for i, p := range points {
		pop[i] = &Particle{
			Id:    i,
			Point: p,
			Best:  optim.NewPoint(p.Pos(), math.Inf(1)),
			Vel:   make([]float64, len(minv)),
		}
		for j := range minv {
			pop[i].Vel[j] = minv[j] + (maxv[j]-minv[j])*optim.RandFloat()
		}
	}
	return pop
}

func (pop Population) Points() []optim.Point {
	points := make([]optim.Point, 0, len(pop))
	for _, p := range pop {
		points = append(points, p.Point)
	}
	return points
}

func (pop Population) Best() optim.Point {
	best := pop[0].Best
	for _, p := range pop[1:] {
		if p.Best.Val < best.Val {
			best = p.Best
		}
	}
	return best
}

type Mover interface {
	Move(p Population)
}

type Iterator struct {
	Pop Population
	optim.Evaler
	Mover
}

func NewIterator(pop Population, e optim.Evaler, m Mover) *Iterator {
	if e == nil {
		e = optim.SerialEvaler{}
	}
	if m == nil {
		m = &SimpleMover{Cognition: DefaultCognition, Social: DefaultSocial}
	}
	return &Iterator{
		Pop:    pop,
		Evaler: optim.SerialEvaler{},
		Mover:  m,
	}
}

func (it Iterator) AddPoint(p optim.Point) {
	if p.Val < it.Pop.Best().Val {
		it.Pop[0].Best = p
	}
}

func (it Iterator) Iterate(obj optim.Objectiver, m mesh.Mesh) (best optim.Point, neval int, err error) {
	points := it.Pop.Points()
	if m != nil {
		for i, p := range points {
			points[i] = optim.Nearest(p, m)
		}
	}
	results, n, err := it.Evaler.Eval(obj, points...)
	if err != nil {
		return optim.Point{}, n, err
	}

	for i := range results {
		it.Pop[i].Update(results[i])
	}

	it.Mover.Move(it.Pop)
	return it.Pop.Best(), n, nil
}

const (
	DefaultCognition = 0.5
	DefaultSocial    = 0.5
	DefaultInertia   = 0.9
)

type SimpleMover struct {
	Cognition float64
	Social    float64
	Vmax      float64
	InertiaFn func() float64
	Rng       *rand.Rand
}

func (mv *SimpleMover) Move(pop Population) {
	if mv.Rng == nil {
		src := rand.NewSource(1)
		mv.Rng = rand.New(src)
	}
	if mv.InertiaFn == nil {
		mv.InertiaFn = func() float64 {
			return DefaultInertia
		}
	}

	best := pop.Best()

	for _, p := range pop {
		vmax := mv.Vmax
		if mv.Vmax == 0 {
			// if no vmax is given, use 1.5 * current speed
			vmax = 1.5 * Speed(p.Vel)
		}

		w1 := mv.Rng.Float64()
		w2 := mv.Rng.Float64()
		// update velocity
		for i, currv := range p.Vel {
			p.Vel[i] = mv.InertiaFn()*currv +
				mv.Cognition*w1*(best.At(i)-p.At(i)) +
				mv.Social*w2*(best.At(i)-p.At(i))
			if s := Speed(p.Vel); mv.Vmax > 0 && Speed(p.Vel) > mv.Vmax {
				for i := range p.Vel {
					p.Vel[i] *= vmax / s
				}
			}

		}

		// update position
		pos := make([]float64, p.Len())
		for i := range pos {
			pos[i] = p.At(i) + p.Vel[i]
		}
		p.Point = optim.NewPoint(pos, p.Val)
	}
}

func Speed(vel []float64) float64 {
	tot := 0.0
	for _, v := range vel {
		tot += v * v
	}
	return math.Sqrt(tot)
}
