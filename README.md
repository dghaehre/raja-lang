# Raja

![Test](https://github.com/dghaehre/raja-lang/actions/workflows/test.yml/badge.svg)

Raja is an expressive, dynamically typed, functional programming language.

It is a small language which uses multiple dispatch and type annototations to make it feel like a statically typed languange.

Here is an example program:

```rust
fizzbuzz = (a:Int) => match [a % 3, a % 5] {
  [0, 0] -> "FizzBuzz"
  [0, _] -> "Fizz"
  [_, 0] -> "Buzz"
  _      -> a.string()
}

range(1, 100).map(fizzbuzz).map(println)
```
