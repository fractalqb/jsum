/*
A tool to analyse the structure of JSON from a set of example JSON values.
Copyright (C) 2025  Marcus Perlick

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package jsum

type Invalid struct {
	error
}

func (Invalid) Accepts(_ any) bool { return false }

func (i Invalid) Example(v any) Deducer { return i }

func (Invalid) Nulls() int { return -1 }

func (Invalid) Hash(dh DedupHash) uint64 { return 0 }

func (Invalid) Equal(_ Deducer) bool { return false }

func (Invalid) Copies() []Deducer { return nil }

func (i Invalid) JSONSchema() any {
	return i.error // TODO
}

var invBase dedBase

func (Invalid) super() *dedBase { return &invBase }
