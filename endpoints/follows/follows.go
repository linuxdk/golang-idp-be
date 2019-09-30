package follows

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"
  "github.com/charmixer/idp/utils"
)

func PostFollows(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostFollows",
    })

    var requests []client.CreateFollowsRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequest = func(iRequests []*utils.Request) {

      var identities []idp.Identity

      createdByIdentityId := c.MustGet("sub").(string)
      identities = append(identities, idp.Identity{Id:createdByIdentityId})

      for _, request := range iRequests {
        if request.Request != nil {
          var r client.CreateFollowsRequest
          r = request.Request.(client.CreateFollowsRequest)
          identities = append(identities, idp.Identity{Id:r.From})
          identities = append(identities, idp.Identity{Id:r.To})
        }
      }

      dbIdentities, err := idp.FetchIdentities(env.Driver, identities)
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }

      var mapIdentities map[string]*idp.Identity
      if ( iRequests[0] != nil ) {
        for _, identity := range dbIdentities {
          mapIdentities[identity.Id] = &identity
        }
      }

      for _, request := range iRequests {
        r := request.Request.(client.CreateFollowsRequest)

        createdBy := mapIdentities[createdByIdentityId]
        if createdBy == nil {
          request.Response = utils.NewClientErrorResponse(request.Index, E.IDENTITY_NOT_FOUND)
          continue
        }

        from := mapIdentities[r.From]
        if from == nil {
          request.Response = utils.NewClientErrorResponse(request.Index, E.IDENTITY_NOT_FOUND)
          continue
        }

        to := mapIdentities[r.To]
        if from == nil {
          request.Response = utils.NewClientErrorResponse(request.Index, E.IDENTITY_NOT_FOUND)
          continue
        }

        follow, err := idp.CreateFollow(env.Driver, *from, *to)
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if follow != (idp.Follow{}) {
          ok := client.Follow{
            From: follow.From.Id,
            To: follow.To.Id,
          }
          var response client.CreateFollowsResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          log.WithFields(logrus.Fields{ "from": follow.From.Id, "to": follow.To.Id }).Debug("Follow created")
          continue
        }

        // Deny by default
        log.WithFields(logrus.Fields{  }).Debug(err.Error())
        request.Response = utils.NewClientErrorResponse(request.Index, E.FOLLOW_NOT_CREATED)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func GetFollows(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetFollows",
    })

    var requests []client.ReadFollowsRequest
    err := c.BindJSON(&requests)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*utils.Request) {

      for _, request := range iRequests {

        var ok []client.Follow

        if request.Request == nil {

          dbFollows, err := idp.FetchFollows(env.Driver, nil)
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          // The empty fetch
          for _, f := range dbFollows {
            ok = append(ok, client.Follow{
              From: f.From.Id,
              To: f.To.Id,
            })
          }
          var response client.ReadFollowsResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          continue

        } else {

          r := request.Request.(client.ReadFollowsRequest)

          dbFollows, err := idp.FetchFollows(env.Driver, []idp.Follow{{From: idp.Identity{Id:r.From}}})
          if err != nil {
            log.WithFields(logrus.Fields{ "id":r.From }).Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          for _, f := range dbFollows {
            ok = append(ok, client.Follow{
              From: f.From.Id,
              To: f.To.Id,
            })
          }
          var response client.ReadFollowsResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          continue
        }

        // Deny by default
        request.Response = utils.NewClientErrorResponse(request.Index, E.FOLLOW_NOT_FOUND)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequests, utils.HandleBulkRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
