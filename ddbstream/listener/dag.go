package listener

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams/types"
)

type dag map[string]*types.Shard

func (d dag) addShards(shards ...*types.Shard) {
	for _, shard := range shards {
		id := aws.ToString(shard.ShardId)
		d[id] = shard
	}
}

func (d dag) Children(id string) (children []*types.Shard) {
	for _, shard := range d {
		if parentID := aws.ToString(shard.ParentShardId); parentID == id {
			children = append(children, shard)
		}
	}
	return children
}

func (d dag) FindAll(conditions ...condition) (shards []*types.Shard) {
loop:
	for _, shard := range d {
		for _, condition := range conditions {
			if !condition(d, shard) {
				continue loop
			}
		}
		shards = append(shards, shard)
	}
	return shards
}

func (d dag) Walk(callback func(shard *types.Shard) error, from ...string) error {
	var shards []*types.Shard
	if len(from) > 0 {
		shards = d.FindAll(startingFrom(d, from...))
	} else {
		shards = d.Roots()
	}

	for _, shard := range shards {
		err := callback(shard)
		if err != nil {
			return err
		}

		err = d.walkID(aws.ToString(shard.ShardId), callback)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d dag) walkID(id string, callback func(shard *types.Shard) error) error {
	for _, shard := range d.Children(id) {
		err := callback(shard)
		if err != nil {
			return err
		}

		err = d.walkID(aws.ToString(shard.ShardId), callback)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d dag) Roots() []*types.Shard {
	return d.FindAll(roots)
}

// condition provides a predicate function to see if
type condition func(dag, *types.Shard) bool

func startingFrom(d dag, roots ...string) condition {
	ancestors := map[string]struct{}{}
	for _, id := range roots {
		if root, ok := d[id]; ok {
			if parentID := aws.ToString(root.ParentShardId); parentID != "" {
				collectAncestors(d, parentID, ancestors)
			}
		}
	}

	return func(d dag, shard *types.Shard) bool {
		got := aws.ToString(shard.ShardId)
		if containsString(roots, got) {
			return true
		}

		if _, ok := ancestors[got]; ok {
			return false
		}

		parentID := aws.ToString(shard.ParentShardId)
		_, hasAncestor := ancestors[parentID]
		if parentID == "" || d[parentID] == nil || hasAncestor {
			return true
		}

		return false
	}
}

func collectAncestors(d dag, id string, ancestors map[string]struct{}) {
	ancestors[id] = struct{}{}
	if shard, ok := d[id]; ok {
		if parentID := aws.ToString(shard.ParentShardId); parentID != "" {
			ancestors[parentID] = struct{}{}
			collectAncestors(d, parentID, ancestors)
		}
	}
}

func roots(d dag, s *types.Shard) bool {
	if parentID := aws.ToString(s.ParentShardId); parentID == "" || d[parentID] == nil {
		return true
	}

	return false
}
