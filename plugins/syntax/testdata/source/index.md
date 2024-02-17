+++
Layout = "page"
+++

# Go #

```go
func fizzbuzz(n int) {
    for i := 1; i <= n; i++ {
        if i%15 == 0 {
            fmt.Println("fizzbuzz")
        } else if i%3 == 0 {
            fmt.Println("fizz")
        } else if i%5 == 0 {
            fmt.Println("buzz")
        } else {
            fmt.Println(i)
        }
    }
}
```

# Python #

```python
def fizzbuzz(n):
    for i in range(1, n + 1):
        if i % 15 == 0:
            print('fizzbuzz')
        elif i % 3 == 0:
            print('fizz')
        elif i % 5 == 0:
            print('buzz')
        else:
            print(i)
```

# JavaScript #

```javascript
function fizzbuzz(n) {
    for (var i = 1; i <= n; i++) {
        if (i % 15 == 0) {
            console.log("fizzbuzz");
        }
        else if (i % 3 == 0) {
            console.log("fizz");
        }
        else if (i % 5 == 0) {
            console.log("buzz");
        }
        else {
            console.log(i);
        }
    }
}
```

# Other #

```
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vestibulum accumsan lacus sit amet tortor vehicula, in
fermentum augue rutrum. Donec eu magna posuere, rhoncus lectus ut, lacinia sapien. Nunc maximus risus risus, ac tempus
augue tristique vitae. Vestibulum consequat, tortor eget auctor viverra, ante elit aliquet purus, quis venenatis ex
velit sit amet neque. Praesent in arcu nec eros interdum placerat. Aliquam mi enim, aliquam consectetur feugiat ut,
euismod sed diam. Ut semper venenatis lobortis. Suspendisse suscipit sem vel tellus fringilla fermentum. Nulla eu odio
justo. Sed rutrum massa commodo nisi laoreet, tincidunt posuere velit gravida. Suspendisse potenti. Praesent varius erat
at leo egestas rhoncus. 
```
