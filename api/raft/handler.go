package raft

import (
	"encoding/json"
	"fmt"
	"github.com/deepmap/oapi-codegen/pkg/gin-middleware"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/raft"
	raftAPI "github.com/subzero112233/golang-raft-nba/api/raft/server"
	"log"
	"net"
	"net/http"
	"time"
)

type Server struct {
	raftServer *raft.Raft
}

func (s Server) GetNodes(c *gin.Context) {
	c.JSON(http.StatusOK, s.raftServer.GetConfiguration().Configuration().Servers)
	return
}

func (s Server) AddNode(c *gin.Context) {
	var node raftAPI.Node
	err := c.ShouldBindJSON(&node)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, raftAPI.ErrorOutput{
			StatusCode: http.StatusBadRequest,
			Message:    "invalid input",
		})

		return
	}

	state := s.raftServer.State()
	if state != raft.Leader {
		address, id := s.raftServer.LeaderWithID()
		c.JSON(http.StatusForbidden, raftAPI.ErrorOutput{
			StatusCode: http.StatusForbidden,
			Message:    fmt.Sprintf("My state is %s. only leaders can join other nodes. leader id is %s and its address is %s", state, id, address),
		})

		return
	}

	if err := isPortOpen(node.Address); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, raftAPI.ErrorOutput{
			StatusCode: http.StatusInternalServerError,
			Message:    fmt.Sprintf("node %s on address %s is unreachable", node.Id, node.Address),
		})

		return
	}

	future := s.raftServer.AddVoter(raft.ServerID(node.Id), raft.ServerAddress(node.Address), 0, 0)
	if future.Error() != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, raftAPI.ErrorOutput{
			StatusCode: http.StatusInternalServerError,
			Message:    fmt.Sprintf("could not add voter with error: %s", err.Error()),
		})

		return
	}

	c.Status(http.StatusOK)
}

func (s Server) RemoveNode(c *gin.Context) {
	var node raftAPI.Node
	err := c.ShouldBindJSON(&node)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, raftAPI.ErrorOutput{
			StatusCode: http.StatusBadRequest,
			Message:    "invalid input",
		})

		return
	}

	state := s.raftServer.State()
	if state != raft.Leader {
		address, id := s.raftServer.LeaderWithID()
		c.JSON(http.StatusForbidden, raftAPI.ErrorOutput{
			StatusCode: http.StatusForbidden,
			Message:    fmt.Sprintf("My state is %s. only leaders can join other nodes. leader id is %s and its address is %s", state, id, address),
		})

		return
	}

	future := s.raftServer.RemoveServer(raft.ServerID(node.Id), 0, 0)
	if future.Error() != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, raftAPI.ErrorOutput{
			StatusCode: http.StatusInternalServerError,
			Message:    fmt.Sprintf("could not remove voter with error: %s", err.Error()),
		})

		return
	}

	c.Status(http.StatusOK)
}

func (s Server) AddStat(c *gin.Context) {
	var event raftAPI.Event
	err := c.ShouldBindJSON(&event)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, raftAPI.ErrorOutput{
			StatusCode: http.StatusBadRequest,
			Message:    "invalid input",
		})

		return
	}

	state := s.raftServer.State()
	if state != raft.Leader {
		address, id := s.raftServer.LeaderWithID()
		c.JSON(http.StatusForbidden, raftAPI.ErrorOutput{
			StatusCode: http.StatusForbidden,
			Message:    fmt.Sprintf("My state is %s. only leaders can join other nodes. leader id is %s and its address is %s", state, id, address),
		})

		return
	}

	eventByte, err := json.Marshal(event)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, raftAPI.ErrorOutput{
			StatusCode: http.StatusBadRequest,
			Message:    err.Error(),
		})
	}

	future := s.raftServer.Apply(eventByte, 0)
	if future.Error() != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, raftAPI.ErrorOutput{
			StatusCode: http.StatusInternalServerError,
			Message:    fmt.Sprintf("could not add voter with error: %s", err.Error()),
		})

		return
	}

	c.Status(http.StatusOK)
}

func StartServer(raftObj *raft.Raft, port string) {
	swagger, err := raftAPI.GetSwagger()
	if err != nil {
		log.Fatal(err)
	}

	// create server and apply request validator
	ginServer := gin.New()
	ginServer.Use(middleware.OapiRequestValidator(swagger))

	raftAPI.RegisterHandlers(ginServer, Server{raftServer: raftObj})
	ginServer.Run(":" + port)
}

func isPortOpen(hostAndPort string) error {
	host, port, err := net.SplitHostPort(hostAndPort)
	if err != nil {
		return err
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Second)
	if err != nil {
		return err
	}

	defer conn.Close()

	return nil
}
