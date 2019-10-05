package disruptor

// Cursors should be a party of the same backing array to keep them as close together as possible:
// https://news.ycombinator.com/item?id=7800825

type Wireup struct {
	capacity int64
	groups   [][]Consumer
	cursors  []*Cursor // backing array keeps cursors (with padding) in contiguous memory
}

func Configure(capacity int64) Wireup {
	return Wireup{
		capacity: capacity,
		groups:   [][]Consumer{},
		cursors:  []*Cursor{NewCursor()},
	}
}

func (this Wireup) WithConsumerGroup(consumers ...Consumer) Wireup {
	if len(consumers) == 0 {
		return this
	}

	target := make([]Consumer, len(consumers))
	copy(target, consumers)

	for i := 0; i < len(consumers); i++ {
		this.cursors = append(this.cursors, NewCursor())
	}

	this.groups = append(this.groups, target)
	return this
}

func (this Wireup) Build() Disruptor {
	var allReaders []*Reader
	var upstream Barrier = this.cursors[0]
	written := this.cursors[0]
	cursorIndex := 1 // 0 index is reserved for the writer Cursor

	for groupIndex, group := range this.groups {
		groupReaders, groupBarrier := this.buildReaders(groupIndex, cursorIndex, written, upstream)
		allReaders = append(allReaders, groupReaders...)
		upstream = groupBarrier
		cursorIndex += len(group)
	}

	writer := NewSingleWriter(written, upstream, this.capacity)
	return Disruptor{writer: writer, readers: allReaders}
}

func (this Wireup) buildReaders(consumerIndex, cursorIndex int, written *Cursor, upstream Barrier) ([]*Reader, Barrier) {
	var barrierCursors []*Cursor
	var readers []*Reader

	for _, consumer := range this.groups[consumerIndex] {
		cursor := this.cursors[cursorIndex]
		barrierCursors = append(barrierCursors, cursor)
		reader := NewReader(cursor, written, upstream, consumer)
		readers = append(readers, reader)
		cursorIndex++
	}

	if len(barrierCursors) == 0 {
		panic("no barriers")
	}

	if len(this.groups[consumerIndex]) == 1 {
		return readers, barrierCursors[0]
	} else {
		return readers, NewCompositeBarrier(barrierCursors)
	}
}
