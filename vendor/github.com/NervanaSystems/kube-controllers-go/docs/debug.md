## Debug short tutorial

- **`make env-up`** bring up the environment

- **`export GODEBUGGER=dlv|gdb`** add your choice of GODEBUGGER to your shell profile
-- See references pages for [gdb](https://golang.org/doc/gdb) and [dlv](https://github.com/derekparker/delve)

- **`make debug`** attach to the streamcontroller process running in the docker container. 
You should see the debugger prompt 
```
Type 'help' for list of commands.
(dlv|gdb)
```

- **`b hooks.go:85`** set a break point in hooks.go

- **`c`** continue running the process

- **`make sp-create`** in a separate terminal create a streamprediction instance

- you should break in the debugger
```
> github.com/NervanaSystems/kube-controllers-go/cmd/stream-prediction-controller/hooks.(*StreamPredictionHooks).addResources() ./hooks/hooks.go:85 (hits goroutine(36):1 total:1) (PC: 0x16f3baa)
    80:		//Delete the resources using name for now.
    81:		h.deleteResources(streamPredict)
    82:	}
    83:
    84:	func (h *StreamPredictionHooks) addResources(streamPredict *crv1.StreamPrediction) error {
=>  85:		for _, resourceClient := range h.resourceClients {
    86:			if err := resourceClient.Create(streamPredict.Namespace(), streamPredict); err != nil {
    87:				glog.Errorf("received err: %v while creating object", err)
    88:				return err
    89:			}
    90:		}
(dlv|gdb)
```
