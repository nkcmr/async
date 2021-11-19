# async go

a prototype of "promises" in go1.18.

**note**: this is just an experiment used to test alternate patterns for dealing with asynchronous code in go. i would not recommend adopting this pattern blindly (especially in production environments) until there is a broader consensus in the go community about this. it might turn out that through some hands-on experience that go's native channels/goroutines are a good enough abstraction and there are no gains to be made from building on top of them.

## install

should be just a regular package:

```
go get -u -v code.nkcmr.net/async@latest
```

## usage

promises abstract away a lot of details about how asynchronous work is handled.
so if you need for something to be async, simply us a promise:

```go
import (
    "context"
    "code.nkcmr.net/async"
)

type MyData struct {/* ... */}

func AsyncFetchData(ctx context.Context, dataID int64) async.Promise[MyData] {
    return async.NewPromise(func() (MyData, error) {
        /* ... */
        return myDataFromRemoteServer, nil
    })
}

func DealWithData(ctx context.Context) {
    myDataPromise := AsyncFetchData(ctx, 451)
    // do other stuff while operation is not settled
    // once your ready to wait for data:
    myData, err := myDataPromise.Await(ctx)
    if err != nil {/* ... */}
}
```
