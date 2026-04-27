package loop

import "time"

type trigger struct {
	lastFire   time.Time
	wordsSince int
}

func newTrigger() *trigger {
	return &trigger{lastFire: time.Now()}
}

func (t *trigger) add(words int) {
	t.wordsSince += words
}

func (t *trigger) shouldFire() bool {
	elapsed := time.Since(t.lastFire)
	return (elapsed >= 6*time.Second && t.wordsSince >= 20) ||
		elapsed >= 18*time.Second ||
		t.wordsSince >= 60
}

func (t *trigger) reset() {
	t.lastFire = time.Now()
	t.wordsSince = 0
}
