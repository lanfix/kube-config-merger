package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

type ConfigNode struct {
	Name       string
	Parameters interface{}
}

type ConfigGroup struct {
    ClustersList   []ConfigNode
    ContextsList   []ConfigNode
    UsersList      []ConfigNode

    // Источник конфигурации для последующего слияния
    Source         string

    // Дополнительные параметры
    CurrentContext string

    // Может ли контент из этой группы быть перезаписан или удален во время мерджа
    CanBeDeleted   bool
}

type ConfigNodePermanent struct {
    ConfigNode   ConfigNode
    CanBeDeleted bool
}

type ArrayFlags []string

func (i *ArrayFlags) String() string {
    return ""
}

func (i *ArrayFlags) Set(value string) error {
    *i = append(*i, value)

    return nil
}

func CastInterfaceToMap(input interface{}) map[string]interface{} {
    return input.(map[string]interface{})
}

func CastInterfaceToList(input interface{}) []interface{} {
    return input.([]interface{})
}

func CollectConfigGroup(configFilePath string) (*ConfigGroup, error) {
    configGroup := ConfigGroup{Source: configFilePath}
	kubeConfigContent, err := os.ReadFile(configGroup.Source)

    if err != nil {
        panic(err)
    }

    if !strings.Contains(string(kubeConfigContent), "apiVersion") {
        return nil, errors.New(fmt.Sprintf("Skipping \"%s\", file does not contain apiVersion", configGroup.Source))
    }

    var kubeConfig interface{}
    err = yaml.Unmarshal(kubeConfigContent, &kubeConfig)

    if err != nil {
        return nil, errors.New(fmt.Sprintf("Skipping \"%s\", invalid yaml syntax", configGroup.Source))
    }

    kubeConfigMapOfInterfaces := CastInterfaceToMap(kubeConfig)
    
    if kubeConfigMapOfInterfaces["apiVersion"].(string) != "v1" ||  kubeConfigMapOfInterfaces["kind"].(string) != "Config" {
        return nil, errors.New(fmt.Sprintf("Skipping \"%s\", file is not a kubernetes access config", configGroup.Source))
    }

    clusterInterfacesList := CastInterfaceToList(kubeConfigMapOfInterfaces["clusters"])
    contextInterfacesList := CastInterfaceToList(kubeConfigMapOfInterfaces["contexts"])
    userInterfacesList := CastInterfaceToList(kubeConfigMapOfInterfaces["users"])

    // Если нет какого-то из параметров (contexts, clusters, users), то нет смысла мерджить такой конфиг
    if len(clusterInterfacesList) == 0 || len(contextInterfacesList) == 0 || len(userInterfacesList) == 0 {
        return nil, errors.New(fmt.Sprintf("Skipping \"%s\", config has not important fields", configGroup.Source))
    }

    configGroup.CurrentContext = kubeConfigMapOfInterfaces["current-context"].(string)

    for _, nodeInterface := range clusterInterfacesList {
        node := CastInterfaceToMap(nodeInterface)
        
        configGroup.ClustersList = append(configGroup.ClustersList, ConfigNode{
            Name: node["name"].(string),
            Parameters: node["cluster"],
        })
    }

    for _, nodeInterface := range contextInterfacesList {
        node := CastInterfaceToMap(nodeInterface)
        
        configGroup.ContextsList = append(configGroup.ContextsList, ConfigNode{
            Name: node["name"].(string),
            Parameters: node["context"],
        })
    }

    for _, nodeInterface := range userInterfacesList {
        node := CastInterfaceToMap(nodeInterface)
        
        configGroup.UsersList = append(configGroup.UsersList, ConfigNode{
            Name: node["name"].(string),
            Parameters: node["user"],
        })
    }

    return &configGroup, nil
}

func RecursiveFilesByDirectory(directoryPath string) []string {
    var fileNames []string

    filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            fmt.Println(err)

            return err
        }

        if !info.IsDir() {
            fileNames = append(fileNames, path)
        }

        return nil
    });

    return fileNames
}

func RecursiveFilesByDirectories(directoryPaths []string) []string {
    var fileNames []string

    for _, directoryPath := range directoryPaths {
        fileNames = append(fileNames, RecursiveFilesByDirectory(directoryPath)...)
    }

    return fileNames
}

func DebugConfigGroups(configGroups []ConfigGroup) {
    fmt.Println("========== Contexts ==========")

    for _, configGroup := range configGroups {
        fmt.Println(configGroup.Source)

        for _, context := range configGroup.ContextsList {
            fmt.Println("---", context.Name)
        }
    }

    fmt.Println("==============================")
}

func GetUniqueString(strings []string) []string {
    uniqueStringsMap := make(map[string]bool)
    var uniqueStrings []string

    for _, item := range strings {
        uniqueStringsMap[item] = true
    }

    for key, _ := range uniqueStringsMap {
        uniqueStrings = append(uniqueStrings, key)
    }

    return uniqueStrings
}

func UnwrapConfigNodesFromMap(configNodes map[string]ConfigNodePermanent) []ConfigNode {
    var output []ConfigNode

    for _, configNodePermanent := range configNodes {
        output = append(output, configNodePermanent.ConfigNode)
    }

    return output
}

func MergeConfigGroups(configGroups []ConfigGroup) (ConfigGroup) {
    mergedClustersMap := make(map[string]ConfigNodePermanent)
    mergedContextsMap := make(map[string]ConfigNodePermanent)
    mergedUsersMap := make(map[string]ConfigNodePermanent)

    // TODO: Отрефакторить
    for _, configGroup := range configGroups {
        for _, configNode := range configGroup.ClustersList {
            if value, ok := mergedClustersMap[configNode.Name]; ok && !value.CanBeDeleted {
                continue;
            }

            mergedClustersMap[configNode.Name] = ConfigNodePermanent{
                ConfigNode: configNode,
                CanBeDeleted: configGroup.CanBeDeleted,
            }
        }
        
        for _, configNode := range configGroup.ContextsList {
            if value, ok := mergedContextsMap[configNode.Name]; ok && !value.CanBeDeleted {
                continue;
            }

            mergedContextsMap[configNode.Name] = ConfigNodePermanent{
                ConfigNode: configNode,
                CanBeDeleted: configGroup.CanBeDeleted,
            }
        }
        
        for _, configNode := range configGroup.UsersList {
            if value, ok := mergedUsersMap[configNode.Name]; ok && !value.CanBeDeleted {
                continue;
            }

            mergedUsersMap[configNode.Name] = ConfigNodePermanent{
                ConfigNode: configNode,
                CanBeDeleted: configGroup.CanBeDeleted,
            }
        }
    }

    return ConfigGroup{
        ClustersList: UnwrapConfigNodesFromMap(mergedClustersMap),
        ContextsList: UnwrapConfigNodesFromMap(mergedContextsMap),
        UsersList: UnwrapConfigNodesFromMap(mergedUsersMap),
    }
}

func (configGroup *ConfigGroup) toYaml() ([]byte, error) {
    var clustersList []interface{}
    var contextsList []interface{}
    var usersList []interface{}

    // TODO: Refactor
    for _, node := range configGroup.ClustersList {
        clustersList = append(clustersList, map[string]interface{}{
            "name": node.Name,
            "cluster": node.Parameters,
        })
    }

    for _, node := range configGroup.ContextsList {
        contextsList = append(contextsList, map[string]interface{}{
            "name": node.Name,
            "context": node.Parameters,
        })
    }

    for _, node := range configGroup.UsersList {
        usersList = append(usersList, map[string]interface{}{
            "name": node.Name,
            "user": node.Parameters,
        })
    }

    configYamlStructure := map[string]interface{}{
        "apiVersion": "v1",
        "kind": "Config",
        "clusters": clustersList,
        "contexts": contextsList,
        "users": usersList,
    }

    if configGroup.CurrentContext != "" {
        configYamlStructure["current-context"] = configGroup.CurrentContext
    }

    return yaml.Marshal(&configYamlStructure)
}

func main() {
    var directories ArrayFlags
    var files ArrayFlags
    var target string

    varbose := flag.Bool("v", false, "Show verbose output of command")
    flag.Var(&directories, "directory", "Specify the directory where configs will be searched")
    flag.Var(&files, "file", "Specify the path to concrete config file")
    flag.StringVar(&target, "target", "~/.kube/config", "Specify the path to the file to merge all the contents there")
    flag.Parse()

    if len(directories) == 0 {
        fmt.Println("You must specify flag --directory and set the name of the directory with configs")

        return
    }

    var configGroups []ConfigGroup
    validFilesToMerge := RecursiveFilesByDirectories(directories)

    for _, filePath := range files {
        if _, err := os.Stat(filePath); err != nil {
            fmt.Println(fmt.Sprintf("Skipping \"%s\", file does not exists", filePath))
            continue
        }

        validFilesToMerge = append(validFilesToMerge, filePath)
    }

    totalValidFilesEdge := 0
    var targetFileMode os.FileMode = 0644

    if info, err := os.Stat(target); err == nil {
        totalValidFilesEdge++
        targetFileMode = info.Mode()

        validFilesToMerge = append(validFilesToMerge, target)
    }

    if len(validFilesToMerge) == totalValidFilesEdge {
        fmt.Println("No valid files to merge")

        return
    }

    // Отбираем только уникальные пути к файлам
    validFilesToMerge = GetUniqueString(validFilesToMerge);

    for _, configureFile := range validFilesToMerge {
        configGroup, err := CollectConfigGroup(configureFile)

        if err != nil {
            fmt.Println(err.Error())

            continue
        }

        configGroup.CanBeDeleted = true

        if (configureFile == target) {
            configGroup.CanBeDeleted = false
        }

        configGroups = append(configGroups, *configGroup)
    }


    if *varbose {
        DebugConfigGroups(configGroups)
    }

    mergedConfig := MergeConfigGroups(configGroups)
    mergedConfig.CurrentContext = configGroups[len(configGroups) - 1].CurrentContext

    yaml, err := mergedConfig.toYaml()

    if err != nil {
        fmt.Println("Can not convert merged structure to yaml")

        return
    }

    err = os.WriteFile(target, yaml, targetFileMode)

    if err != nil {
        fmt.Println("Can not write target file")
    }
}
