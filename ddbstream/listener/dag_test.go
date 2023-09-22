package listener

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams/types"
)

func newShard(id, parentID string) *types.Shard {
	shard := &types.Shard{
		ShardId: aws.String(id),
	}
	if parentID != "" {
		shard.ParentShardId = aws.String(parentID)
	}
	return shard
}
func Test_Roots(t *testing.T) {
	var (
		a   = newShard("A", "")
		a1  = newShard("A1", "A")
		a1a = newShard("A1A", "A1")
		a2  = newShard("A2", "A")
		b   = newShard("B", "")
	)

	testCases := map[string]struct {
		Shards []*types.Shard
		Roots  []string
		Want   []string
	}{
		"nop": {},
		"all": {
			Shards: []*types.Shard{a, a1, a1a, a2, b},
			Roots:  nil,
			Want:   []string{"a", "b"},
		},
		"a is explicit, b is not": {
			Shards: []*types.Shard{a1a, a2},
			Roots:  nil,
			Want:   []string{"a", "b"},
		},
	}

	for label, tc := range testCases {
		t.Run(label, func(t *testing.T) {
			d := dag{}
			d.addShards(tc.Shards...)
			d.FindAll(startingFrom(d, tc.Roots...))
		})
	}
}
