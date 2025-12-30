export class Registry<T> {
  private items_: T[];
  private seen_: Set<T>;

  constructor() {
    this.items_ = [];
    this.seen_ = new Set();
  }

  register(t: T): void {
    if (!this.seen_.has(t)) {
      this.items_.push(t);
      this.seen_.add(t);
    }
  }

  get items(): T[] {
    return this.items_.slice();
  }
}
