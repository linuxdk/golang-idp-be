package identities

import (
  "fmt"
  "net/http"

  "github.com/gin-gonic/gin"

  "golang-idp-be/config"
  "golang-idp-be/gateway/idpbe"
  "golang-idp-be/gateway/hydra"
)

type AuthenticateRequest struct {
  Id              string            `json:"id"`
  Password        string            `json:"password"`
  Challenge       string            `json:"challenge" binding:"required"`
}

type AuthenticateResponse struct {
  Id              string            `json:"id"`
  Authenticated   bool              `json:"authenticated"`
}

func PostAuthenticate(env *idpbe.IdpBeEnv) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    fmt.Println(fmt.Sprintf("[request-id:%s][event:identities.PostAuthenticate]", c.MustGet("RequestId")))

    var input AuthenticateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    // Create a new HTTP client to perform the request, to prevent serialization
    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    hydraLoginResponse, err := hydra.GetLogin(config.Hydra.LoginRequestUrl, hydraClient, input.Challenge)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return;
    }

    if hydraLoginResponse.Skip {
      hydraLoginAcceptRequest := hydra.HydraLoginAcceptRequest{
        Subject: hydraLoginResponse.Subject,
        Remember: true,
        RememberFor: 30,
      }

      hydraLoginAcceptResponse := hydra.AcceptLogin(config.Hydra.LoginRequestAcceptUrl, hydraClient, input.Challenge, hydraLoginAcceptRequest)

      fmt.Println("IdpBe.PostIdentitiesAuthenticate, id:"+input.Id+" authenticated:true redirect_to:"+hydraLoginAcceptResponse.RedirectTo)
      c.JSON(http.StatusOK, gin.H{
        "id": input.Id,
        "authenticated": true,
        "redirect_to": hydraLoginAcceptResponse.RedirectTo,
      })
      c.Abort()
      return
    }

    dbIdentity := env.Database[input.Id]
    if dbIdentity != nil {

      // FIXME: Implement password algorithm in storage and hash check.
      fmt.Println("IdpBe.PostIdentitiesAuthenticate, Password check is stupid and for testing use a real pw hash")
      if dbIdentity.Id == input.Id && dbIdentity.Password == input.Password {
        hydraLoginAcceptRequest := hydra.HydraLoginAcceptRequest{
          Subject: dbIdentity.Id,
          Remember: true,
          RememberFor: 30,
        }

        hydraLoginAcceptResponse := hydra.AcceptLogin(config.Hydra.LoginRequestAcceptUrl, hydraClient, input.Challenge, hydraLoginAcceptRequest)

        fmt.Println("IdpBe.PostIdentitiesAuthenticate, id:"+dbIdentity.Id+" authenticated:true redirect_to:"+hydraLoginAcceptResponse.RedirectTo)
        c.JSON(http.StatusOK, gin.H{
          "id": dbIdentity.Id,
          "authenticated": true,
          "redirect_to": hydraLoginAcceptResponse.RedirectTo,
        })
        c.Abort()
        return
      }

    }

    // Deny by default
    fmt.Println("IdpBe.PostIdentitiesAuthenticate, id:"+input.Id+" authenticated:false redirect_to:")
    c.JSON(http.StatusOK, gin.H{
      "id": input.Id,
      "authenticated": false,
    })
  }
  return gin.HandlerFunc(fn)
}
