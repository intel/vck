Controller Development Guidelines
=================================

Below you will find some guidelines to keep in mind when developing a controller.

## Idempotence

__Idempotence__: _The property of certain operations in mathematics and computer science that they can be applied multiple times without changing the result beyond the initial application<sup>[1]</sup>_

### In a Controller
Considering the idempotence of the actions we take against any shared resource allows for the running of multiple instances of a controller. This has strong positive effects on uptime. Ostensibly, the reason to have more than one instance of a controller running is for the sake of resilience, but it also simplifies rolling upgrades: An instance with a newer version could be started while the older instance is torn down without any downtime.

#### Example: The API Server is Trustworthy

![a tale of n controllers](https://user-images.githubusercontent.com/1194436/31556517-3d923c42-affa-11e7-8fa7-e8ad623570d6.png)

One of the reasons for writing idempotent controllers comes from the manner in which events are delivered.  The API Server uses SSE as an event delivery mechanism, which can only work by delivering events to all subscribers of some source.  In order to mitigate this in an idempotent manner, we can do two things, each of which hinges on a guarantee from the API Server that it is sequentially consistent<sup>[2]</sup>.

__1. Take responsibility for and have some deterministic manner in which to generate identifiers.__

When a new resource is created, take something from the Nervana Cloud domain like `stream_id` and couple it with a fixed descriptor like `krypton-sp13` rather than randomly generating identifiers.  This results in any instance of a controller generating the same name for a resource.

__2. Gracefully handle an "already exists" error.__

Coupled with number 1, if two controllers attempted to create the resource `krypton-sp13` only one would succeed and the API Server would reply to the second with an error that `krypton-sp13` already exists.  The controller should inspect the returned error from a creation request and treat "already exists" as a special case, one that can generally be dismissed.  Don't forget logging here either!  The right choice is `INFO` or at most `WARN`; be wary of using `ERROR` in this case as it can be misleading. The goal of the logging should be to report the possible competition of a set of controller instances, not that the resource already exists.

1. https://en.wikipedia.org/wiki/Idempotence
2. https://en.wikipedia.org/wiki/Sequential_consistency
