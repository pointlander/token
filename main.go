// Copyright 2020 The Token Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"syscall"
)

// Size is the size of the population
const Size = 100

// Curie is the wiki on curie
var Curie []byte

// Genome is a token genome
type Genome struct {
	Tokens  []int64
	Fitness float64
}

// NewGenome creates a new genome
func NewGenome() Genome {
	length := len(Curie)
	tokens := make([]int64, length)
	for i := range tokens {
		tokens[i] = int64(rand.Intn(length))
	}
	return Genome{
		Tokens: tokens,
	}
}

// ComputeFitness computes the fitness of the genome
func (g *Genome) ComputeFitness() {
	tokens := make(map[int64][]byte)
	for i, token := range g.Tokens {
		t := tokens[token]
		if t == nil {
			t = make([]byte, 0, 8)
		}
		t = append(t, Curie[i])
		tokens[token] = t
	}

	fitness, length := 0.0, len(tokens)
	for _, set := range tokens {
		buffer := bytes.Buffer{}
		writer := gzip.NewWriter(&buffer)
		_, err := writer.Write(set)
		if err != nil {
			panic(err)
		}
		err = writer.Close()
		if err != nil {
			panic(err)
		}
		fitness += float64(buffer.Len()) / float64(len(set))
	}

	buffer := bytes.Buffer{}
	writer := gzip.NewWriter(&buffer)
	output := make([]byte, 8)
	for _, t := range g.Tokens {
		binary.LittleEndian.PutUint64(output, uint64(t))
		_, err := writer.Write(output)
		if err != nil {
			panic(err)
		}
	}
	err := writer.Close()
	if err != nil {
		panic(err)
	}
	fitness += float64(length*buffer.Len()) / (float64(len(g.Tokens)) * math.Log2(float64(length)))

	g.Fitness = fitness
}

// Copy copies a genome
func (g *Genome) Copy() Genome {
	tokens := make([]int64, len(g.Tokens))
	copy(tokens, g.Tokens)
	return Genome{
		Tokens: tokens,
	}
}

// Print prints the genome
func (g *Genome) Print() {
	tokens := make(map[int64][]byte)
	for i, token := range g.Tokens {
		t := tokens[token]
		if t == nil {
			t = make([]byte, 0, 8)
		}
		t = append(t, Curie[i])
		tokens[token] = t
	}

	for key, value := range tokens {
		fmt.Println(key, string(value))
	}
}

func main() {
	rand.Seed(1)

	input, err := ioutil.ReadFile("curie.wiki")
	if err != nil {
		panic(err)
	}
	Curie = input[:1024]

	genomes := make([]Genome, 0, Size)
	for i := 0; i < Size; i++ {
		genome := NewGenome()
		genomes = append(genomes, genome)
	}

	fini, exit := false, make(chan os.Signal)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-exit
		fmt.Println("exit")
		fini = true
	}()

	for {
		done := make(chan int, 8)
		fitness := func(i int) {
			genomes[i].ComputeFitness()
			done <- i
		}
		for i := range genomes {
			go fitness(i)
		}
		for range genomes {
			<-done
		}
		sort.Slice(genomes, func(i, j int) bool {
			return genomes[i].Fitness < genomes[j].Fitness
		})
		genomes = genomes[:Size]
		tokens := make(map[int64]bool)
		for _, t := range genomes[0].Tokens {
			tokens[t] = true
		}
		fmt.Println(genomes[0].Fitness, len(tokens))

		if genomes[0].Fitness < 4050 || fini {
			genomes[0].Print()
			break
		}

		for i := 0; i < Size; i++ {
			switch rand.Intn(3) {
			case 0:
				a := rand.Intn(10)
				cp := genomes[a].Copy()
				mutate := rand.Intn(len(cp.Tokens))
				switch rand.Intn(2) {
				case 0:
					cp.Tokens[mutate]++
					if length := int64(len(Curie) - 1); cp.Tokens[mutate] > length {
						cp.Tokens[mutate] = length
					}
				case 1:
					cp.Tokens[mutate]--
					if cp.Tokens[mutate] < 0 {
						cp.Tokens[mutate] = 0
					}
				}
				genomes = append(genomes, cp)
			case 1:
				a, b := rand.Intn(10), rand.Intn(10)
				cpa, cpb := genomes[a].Copy(), genomes[b].Copy()
				x, y := rand.Intn(len(cpa.Tokens)), rand.Intn(len(cpb.Tokens))
				cpa.Tokens[x], cpb.Tokens[y] = cpb.Tokens[y], cpa.Tokens[x]
				genomes = append(genomes, cpa, cpb)
			case 2:
				a, b := rand.Intn(10), rand.Intn(10)
				cpa, cpb := genomes[a].Copy(), genomes[b].Copy()
				x, y := rand.Intn(len(cpa.Tokens)), rand.Intn(len(cpb.Tokens))
				cpa.Tokens[x] = cpb.Tokens[y]
				genomes = append(genomes, cpa, cpb)
			}
		}
	}
}
