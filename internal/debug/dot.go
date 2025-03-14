// Copyright © 2018 Heptio
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package debug

import (
	"fmt"
	"io"

	"github.com/heptio/contour/internal/dag"
	"github.com/heptio/contour/internal/envoy"
)

// quick and dirty dot debugging package

type dotWriter struct {
	kc                    *dag.KubernetesCache
	disablePermitInsecure bool
}

type pair struct {
	a, b interface{}
}

type ctx struct {
	w     io.Writer
	nodes map[interface{}]bool
	edges map[pair]bool
}

func (c *ctx) writeVertex(v dag.Vertex) {
	if c.nodes[v] {
		return
	}
	c.nodes[v] = true
	switch v := v.(type) {
	case *dag.Listener:
		fmt.Fprintf(c.w, `"%p" [shape=record, label="{listener|%s:%d}"]`+"\n", v, v.Address, v.Port)
	case *dag.Secret:
		fmt.Fprintf(c.w, `"%p" [shape=record, label="{secret|%s/%s}"]`+"\n", v, v.Namespace(), v.Name())
	case *dag.HTTPService:
		fmt.Fprintf(c.w, `"%p" [shape=record, label="{httpservice|%s/%s:%d}"]`+"\n", v, v.Namespace, v.Name, v.Port)
	case *dag.VirtualHost:
		fmt.Fprintf(c.w, `"%p" [shape=record, label="{http://%s}"]`+"\n", v, v.Name)
	case *dag.SecureVirtualHost:
		fmt.Fprintf(c.w, `"%p" [shape=record, label="{https://%s}"]`+"\n", v, v.VirtualHost.Name)
	case *dag.PrefixRoute:
		fmt.Fprintf(c.w, `"%p" [shape=record, label="{prefix|%s}"]`+"\n", v, v.Prefix)
	case *dag.RegexRoute:
		fmt.Fprintf(c.w, `"%p" [shape=record, label="{regex|%s}"]`+"\n", v, v.Regex)
	case *dag.TCPService:
		fmt.Fprintf(c.w, `"%p" [shape=record, label="{tcpservice|%s/%s:%d}"]`+"\n", v, v.Namespace, v.Name, v.Port)
	case *dag.TCPProxy:
		fmt.Fprintf(c.w, `"%p" [shape=record, label="{tcpproxy}"]`+"\n", v)
	case *dag.Cluster:
		fmt.Fprintf(c.w, `"%p" [shape=record, label="{cluster|{%s|weight %d}}"]`+"\n", v, envoy.Clustername(v), v.Weight)
	}
}

func (c *ctx) writeEdge(parent, child dag.Vertex) {
	if c.edges[pair{parent, child}] {
		return
	}
	c.edges[pair{parent, child}] = true
	fmt.Fprintf(c.w, `"%p" -> "%p"`+"\n", parent, child)
}

func (dw *dotWriter) writeDot(w io.Writer) {
	fmt.Fprintln(w, "digraph DAG {\nrankdir=\"LR\"")

	ctx := &ctx{
		w:     w,
		nodes: make(map[interface{}]bool),
		edges: make(map[pair]bool),
	}

	var visit func(dag.Vertex)
	visit = func(parent dag.Vertex) {
		ctx.writeVertex(parent)
		parent.Visit(func(child dag.Vertex) {
			visit(child)
			ctx.writeEdge(parent, child)
		})
	}

	builder := &dag.Builder{
		Source:                dw.kc,
		DisablePermitInsecure: dw.disablePermitInsecure,
	}
	builder.Build().Visit(visit)

	fmt.Fprintln(w, "}")
}
