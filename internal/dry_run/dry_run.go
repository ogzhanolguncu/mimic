package dryrun

import (
	"fmt"
	"log"
	"strings"

	"github.com/ogzhanolguncu/mimic/internal/syncer"
)

type Node struct {
	fileName   string
	fileSize   int
	actionType int
	children   *[]Node
}

func printSummary(stats map[int]struct {
	Count int
	Size  int64
},
) {
	log.Printf("==== DRY RUN MODE: No changes will be made ====\n")
	log.Printf("SUMMARY OF ACTIONS:\n")
	log.Printf("* Files to create: %d (total size: %.1f MB)\n",
		stats[syncer.ActionCreate].Count,
		float64(stats[syncer.ActionCreate].Size)/(1024*1024))
	log.Printf("* Files to update: %d (total size: %.1f MB)\n",
		stats[syncer.ActionUpdate].Count,
		float64(stats[syncer.ActionUpdate].Size)/(1024*1024))
	log.Printf("* Files to delete: %d (total size: %.1f MB)\n",
		stats[syncer.ActionDelete].Count,
		float64(stats[syncer.ActionDelete].Size)/(1024*1024))
	log.Printf("* Directories to create: %d\n", stats[syncer.ActionCreate|0x10].Count) // Assuming flag for directories
	log.Printf("* Directories to delete: %d\n", stats[syncer.ActionDelete|0x10].Count) // Assuming flag for directories
	log.Printf("* Unchanged: %d\n", stats[syncer.ActionNone].Count)
}

func printTree(node *Node, indent string) {
	if node == nil {
		return
	}

	var actionStr string
	switch node.actionType {
	case syncer.ActionNone:
		actionStr = "NONE"
	case syncer.ActionCreate:
		actionStr = "CREATE"
	case syncer.ActionUpdate:
		actionStr = "UPDATE"
	case syncer.ActionDelete:
		actionStr = "DELETE"
	default:
		actionStr = "UNKNOWN"
	}

	// Format file size
	var sizeStr string
	if node.fileSize < 1024 {
		sizeStr = fmt.Sprintf("%d B", node.fileSize)
	} else if node.fileSize < 1024*1024 {
		sizeStr = fmt.Sprintf("%.1f KB", float64(node.fileSize)/1024)
	} else if node.fileSize < 1024*1024*1024 {
		sizeStr = fmt.Sprintf("%.1f MB", float64(node.fileSize)/(1024*1024))
	} else {
		sizeStr = fmt.Sprintf("%.1f GB", float64(node.fileSize)/(1024*1024*1024))
	}

	// Print current node with action type and file size
	log.Printf("%s- %s [%s] (%s)", indent, node.fileName, actionStr, sizeStr)

	// Print children recursively with increased indentation
	if node.children != nil {
		for i := range *node.children {
			printTree(&(*node.children)[i], indent+"  ")
		}
	}
}

func PrintFullReport(actions []syncer.SyncAction) {
	rootNode := generateTree(actions)
	// First gather statistics
	stats := collectStats(&rootNode)

	// Print summary
	printSummary(stats)

	// Print detailed tree
	printTree(&rootNode, "")
}

func generateTree(actions []syncer.SyncAction) Node {
	rootNode := Node{fileName: "(root)", children: &[]Node{}}

	for _, action := range actions {
		path := action.RelativePath
		// Trim any leading slash
		path = strings.TrimPrefix(path, "/")
		// Split the path into components
		components := strings.Split(path, "/")

		currentNode := &rootNode

		// Navigate/create the path
		for _, component := range components {
			// Find if a child with this name already exists
			var found bool
			var nextNode *Node

			if currentNode.children != nil {
				for i := range *currentNode.children {
					if (*currentNode.children)[i].fileName == component {
						nextNode = &(*currentNode.children)[i]
						found = true
						break
					}
				}
			}

			// If not found, create a new node and add it as a child
			if !found {
				// Initialize children slice if needed
				if currentNode.children == nil {
					emptySlice := make([]Node, 0)
					currentNode.children = &emptySlice
				}

				// Create new node
				newNode := Node{
					fileName:   component,
					children:   nil,
					actionType: action.Type,
					fileSize:   int(action.SourceInfo.Size),
				}

				// Add to children
				newChildren := append(*currentNode.children, newNode)
				currentNode.children = &newChildren

				// Get reference to the newly added node
				nextNode = &(*currentNode.children)[len(*currentNode.children)-1]
			}

			// Move to the next level
			currentNode = nextNode
		}
	}
	return rootNode
}

func collectStats(node *Node) map[int]struct {
	Count int
	Size  int64
} {
	stats := make(map[int]struct {
		Count int
		Size  int64
	})

	collectStatsRecursive(node, stats)

	return stats
}

func collectStatsRecursive(node *Node, stats map[int]struct {
	Count int
	Size  int64
},
) {
	if node == nil {
		return
	}

	// Update stats for current node
	statEntry := stats[node.actionType]
	statEntry.Count++
	statEntry.Size += int64(node.fileSize)
	stats[node.actionType] = statEntry

	// Process children
	if node.children != nil {
		for i := range *node.children {
			collectStatsRecursive(&(*node.children)[i], stats)
		}
	}
}
