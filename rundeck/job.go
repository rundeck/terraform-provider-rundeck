// The MIT License (MIT)

// Copyright (c) 2015 Martin Atkins

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package rundeck

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/rundeck/go-rundeck/rundeck"
)

// JobSummary is an abbreviated description of a job that includes only its basic
// descriptive information and identifiers.
type JobSummary struct {
	XMLName     xml.Name `xml:"job"`
	ID          string   `xml:"id,attr"`
	Name        string   `xml:"name"`
	GroupName   string   `xml:"group"`
	ProjectName string   `xml:"project"`
	Description string   `xml:"description,omitempty"`
}

/*
type jobSummaryList struct {
	XMLName xml.Name     `xml:"jobs"`
	Jobs    []JobSummary `xml:"job"`
}
*/

// JobDetail is a comprehensive description of a job, including its entire definition.
type JobDetail struct {
	XMLName                   xml.Name            `xml:"job"`
	ID                        string              `xml:"uuid,omitempty"`
	Name                      string              `xml:"name"`
	GroupName                 string              `xml:"group,omitempty"`
	ProjectName               string              `xml:"context>project,omitempty"`
	OptionsConfig             *JobOptions         `xml:"context>options,omitempty"`
	Description               string              `xml:"description"`
	ExecutionEnabled          bool                `xml:"executionEnabled"`
	LogLevel                  string              `xml:"loglevel,omitempty"`
	AllowConcurrentExecutions bool                `xml:"multipleExecutions,omitempty"`
	Dispatch                  *JobDispatch        `xml:"dispatch,omitempty"`
	CommandSequence           *JobCommandSequence `xml:"sequence,omitempty"`
	Notification              *JobNotification    `xml:"notification,omitempty"`
	Timeout                   string              `xml:"timeout,omitempty"`
	Retry                     string              `xml:"retry,omitempty"`
	NodeFilter                *JobNodeFilter      `xml:"nodefilters,omitempty"`

	/* If Dispatch is enabled, nodesSelectedByDefault is always present with true/false.
	 * by this reason omitempty cannot be present.
	 * This has to be handle by the user.
	 */
	NodesSelectedByDefault *Boolean     `xml:"nodesSelectedByDefault"`
	Schedule               *JobSchedule `xml:"schedule,omitempty"`
	ScheduleEnabled        bool         `xml:"scheduleEnabled"`
	TimeZone               string       `xml:"timeZone,omitempty"`
}

type Boolean struct {
	Value bool `xml:",chardata"`
}

type JobNotification struct {
	OnFailure *Notification `xml:"onfailure,omitempty"`
	OnStart   *Notification `xml:"onstart,omitempty"`
	OnSuccess *Notification `xml:"onsuccess,omitempty"`
}

type Notification struct {
	Email   *EmailNotification   `xml:"email,omitempty"`
	WebHook *WebHookNotification `xml:"webhook,omitempty"`
	Plugin  *JobPlugin           `xml:"plugin"`
}

type EmailNotification struct {
	AttachLog  bool               `xml:"attachLog,attr,omitempty"`
	Recipients NotificationEmails `xml:"recipients,attr"`
	Subject    string             `xml:"subject,attr"`
}

type NotificationEmails []string

type WebHookNotification struct {
	Urls       NotificationUrls `xml:"urls,attr"`
	HttpMethod string           `xml:"httpMethod,attr"`
}

type NotificationUrls []string

type JobSchedule struct {
	XMLName xml.Name           `xml:"schedule"`
	Time    JobScheduleTime    `xml:"time"`
	Month   JobScheduleMonth   `xml:"month"`
	WeekDay JobScheduleWeekDay `xml:"weekday"`
	Year    JobScheduleYear    `xml:"year"`
}

type JobScheduleMonth struct {
	XMLName xml.Name `xml:"month"`
	Day     string   `xml:"day,attr"`
	Month   string   `xml:"month,attr"`
}

type JobScheduleYear struct {
	XMLName xml.Name `xml:"year"`
	Year    string   `xml:"year,attr"`
}

type JobScheduleWeekDay struct {
	XMLName xml.Name `xml:"weekday"`
	Day     string   `xml:"day,attr"`
}

type JobScheduleTime struct {
	XMLName xml.Name `xml:"time"`
	Hour    string   `xml:"hour,attr"`
	Minute  string   `xml:"minute,attr"`
	Seconds string   `xml:"seconds,attr"`
}

type jobDetailList struct {
	XMLName xml.Name    `xml:"joblist"`
	Jobs    []JobDetail `xml:"job"`
}

// JobOptions represents the set of options on a job, if any.
type JobOptions struct {
	PreserveOrder bool        `xml:"preserveOrder,attr,omitempty"`
	Options       []JobOption `xml:"option"`
}

// JobOption represents a single option on a job.
type JobOption struct {
	XMLName xml.Name `xml:"option"`

	// If AllowsMultipleChoices is set, the string that will be used to delimit the multiple
	// chosen options.
	MultiValueDelimiter string `xml:"delimiter,attr,omitempty"`

	// If set, Rundeck will reject values that are not in the set of predefined choices.
	RequirePredefinedChoice bool `xml:"enforcedvalues,attr,omitempty"`

	// When either ValueChoices or ValueChoicesURL is set, controls whether more than one
	// choice may be selected as the value.
	AllowsMultipleValues bool `xml:"multivalued,attr,omitempty"`

	// The name of the option, which can be used to interpolate its value
	// into job commands.
	Name string `xml:"name,attr,omitempty"`

	// The displayed label of the option.
	Label string `xml:"label,omitempty"`

	// Regular expression to be used to validate the option value.
	ValidationRegex string `xml:"regex,attr,omitempty"`

	// If set, Rundeck requires a value to be set for this option.
	IsRequired bool `xml:"required,attr,omitempty"`

	// If set, the input for this field will be obscured in the UI. Useful for passwords
	// and other secrets.
	ObscureInput bool `xml:"secure,attr,omitempty"`

	// If ObscureInput is set, StoragePath can be used to point out credentials.
	StoragePath string `xml:"storagePath,attr,omitempty"`

	// The default value of the option.
	DefaultValue string `xml:"value,attr,omitempty"`

	// If set, the value can be accessed from scripts.
	ValueIsExposedToScripts bool `xml:"valueExposed,attr,omitempty"`

	// A sequence of predefined choices for this option. Mutually exclusive with ValueChoicesURL.
	ValueChoices JobValueChoices `xml:"values,attr"`

	// A URL from which the predefined choices for this option will be retrieved.
	// Mutually exclusive with ValueChoices
	ValueChoicesURL string `xml:"valuesUrl,attr,omitempty"`

	// Description of the value to be shown in the Rundeck UI.
	Description string `xml:"description,omitempty"`
}

// JobValueChoices is a specialization of []string representing a sequence of predefined values
// for a job option.
type JobValueChoices []string

// JobCommandSequence describes the sequence of operations that a job will perform.
type JobCommandSequence struct {
	XMLName xml.Name `xml:"sequence"`

	// If set, Rundeck will continue with subsequent commands after a command fails.
	ContinueOnError bool `xml:"keepgoing,attr,omitempty"`

	// Chooses the strategy by which Rundeck will execute commands. Can either be "node-first" or
	// "step-first".
	OrderingStrategy string `xml:"strategy,attr,omitempty"`

	// Log outputs to be captured and used across command sequence
	GlobalLogFilters *[]JobLogFilter `xml:"pluginConfig>LogFilter,omitempty"`

	// Sequence of commands to run in the sequence.
	Commands []JobCommand `xml:"command"`

	// Description
	Description string `xml:"description,omitempty"`
}

// JobCommand describes a particular command to run within the sequence of commands on a job.
// The members of this struct are mutually-exclusive except for the pair of ScriptFile and
// ScriptFileArgs.
type JobCommand struct {
	XMLName xml.Name

	// Description
	Description string `xml:"description,omitempty"`

	// If the Workflow keepgoing is false, this allows the Workflow to continue when the Error Handler is successful.
	KeepGoingOnSuccess bool `xml:"keepgoingOnSuccess,attr,omitempty"`

	// On error:
	ErrorHandler *JobCommand `xml:"errorhandler,omitempty"`

	// A literal shell command to run.
	ShellCommand string `xml:"exec,omitempty"`

	// Add extension to the temporary filename.
	FileExtension string `xml:"fileExtension,omitempty"`

	// An inline program to run. This will be written to disk and executed, so if it is
	// a shell script it should have an appropriate #! line.
	Script string `xml:"script,omitempty"`

	// A pre-existing file (on the target nodes) that will be executed.
	ScriptFile string `xml:"scriptfile,omitempty"`

	// When ScriptFile is set, the arguments to provide to the script when executing it.
	ScriptFileArgs string `xml:"scriptargs,omitempty"`

	// ScriptInterpreter is used to execute ScriptFile.
	ScriptInterpreter *JobCommandScriptInterpreter `xml:"scriptinterpreter,omitempty"`

	// A reference to another job to run as this command.
	Job *JobCommandJobRef `xml:"jobref"`

	// Configuration for a step plugin to run as this command.
	StepPlugin *JobPlugin `xml:"step-plugin"`

	// Configuration for a node step plugin to run as this command.
	NodeStepPlugin *JobPlugin `xml:"node-step-plugin"`
}

// (Inline) Script interpreter
type JobCommandScriptInterpreter struct {
	XMLName          xml.Name `xml:"scriptinterpreter"`
	InvocationString string   `xml:",chardata"`
	ArgsQuoted       bool     `xml:"argsquoted,attr,omitempty"`
}

// JobCommandJobRef is a reference to another job that will run as one of the commands of a job.
type JobCommandJobRef struct {
	XMLName        xml.Name                  `xml:"jobref"`
	Name           string                    `xml:"name,attr"`
	GroupName      string                    `xml:"group,attr"`
	RunForEachNode bool                      `xml:"nodeStep,attr"`
	Dispatch       *JobDispatch              `xml:"dispatch,omitempty"`
	NodeFilter     *JobNodeFilter            `xml:"nodefilters,omitempty"`
	Arguments      JobCommandJobRefArguments `xml:"arg"`
}

// JobCommandJobRefArguments is a string representing the arguments in a JobCommandJobRef.
type JobCommandJobRefArguments string

// Plugin is a configuration for a plugin to run within a job or notification.
type JobPlugin struct {
	XMLName xml.Name
	Type    string          `xml:"type,attr"`
	Config  JobPluginConfig `xml:"configuration"`
}

// JobPluginConfig is a specialization of map[string]string for job plugin configuration.
type JobPluginConfig map[string]string

// JobNodeFilter describes which nodes from the project's resource list will run the configured
// commands.
type JobNodeFilter struct {
	Query             string `xml:"filter,omitempty"`
	ExcludeQuery      string `xml:"filterExclude,omitempty"`
	ExcludePrecedence bool   `xml:"excludeprecedence,omitempty"`
}

type jobImportResults struct {
	Succeeded jobImportResultsCategory `xml:"succeeded"`
	Failed    jobImportResultsCategory `xml:"failed"`
	Skipped   jobImportResultsCategory `xml:"skipped"`
}

type jobImportResultsCategory struct {
	Count   int               `xml:"count,attr"`
	Results []jobImportResult `xml:"job"`
}

type jobImportResult struct {
	ID          string `xml:"id,omitempty"`
	Name        string `xml:"name"`
	GroupName   string `xml:"group,omitempty"`
	ProjectName string `xml:"context>project,omitempty"`
	Error       string `xml:"error"`
}

type JobLogFilter struct {
	XMLName xml.Name            `xml:"LogFilter"`
	Type    string              `xml:"type,attr"`
	Config  *JobLogFilterConfig `xml:"config,omitempty"`
}

// JobLogFilterConfig is a specialization of map[string]string for job log filter configuration.
type JobLogFilterConfig map[string]string

type xmlJobLogFilterConfigEntry struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type JobDispatch struct {
	MaxThreadCount           int    `xml:"threadcount,omitempty"`
	ContinueNextNodeOnError  bool   `xml:"keepgoing,omitempty"`
	RankAttribute            string `xml:"rankAttribute,omitempty"`
	RankOrder                string `xml:"rankOrder,omitempty"`
	SuccessOnEmptyNodeFilter bool   `xml:"successOnEmptyNodeFilter,omitempty"`
}

// GetJobSummariesForProject returns summaries of the jobs belonging to the named project.
// func (c *Client) GetJobSummariesForProject(projectName string) ([]JobSummary, error) {
// 	jobList := &jobSummaryList{}
// 	err := c.get([]string{"project", projectName, "jobs"}, nil, jobList)
// 	return jobList.Jobs, err
// }

// GetJobsForProject returns the full job details of the jobs belonging to the named project.
// func (c *Client) GetJobsForProject(projectName string) ([]JobDetail, error) {
// 	jobList := &jobDetailList{}
// 	err := c.get([]string{"jobs", "export"}, map[string]string{"project": projectName}, jobList)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return jobList.Jobs, nil
// }

// GetJob returns the full job details of the job with the given id.
func GetJob(c *rundeck.BaseClient, id string) (*JobDetail, error) {
	ctx := context.Background()
	jobList := &jobDetailList{}
	resp, err := c.JobGet(ctx, id, "")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, &NotFoundError{}
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error getting job: (%v)", err)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := xml.Unmarshal(respBytes, jobList); err != nil {
		return nil, err
	}

	return &jobList.Jobs[0], nil
}

// CreateJob creates a new job based on the provided structure.
// func CreateJob(c *rundeck.BaseClient,job *JobDetail) (*JobSummary, error) {
// 	return importJob(c, job, "create")
// }

// CreateOrUpdateJob takes a job detail structure which has its ID set and either updates
// an existing job with the same id or creates a new job with that id.
// func (c *Client) CreateOrUpdateJob(job *JobDetail) (*JobSummary, error) {
// 	return c.importJob(job, "update")
// }

func importJob(c *rundeck.BaseClient, job *JobDetail, dupeOption string) (*JobSummary, error) {
	jobList := &jobDetailList{
		Jobs: []JobDetail{*job},
	}
	// args := map[string]string{
	// 	"format":     "xml",
	// 	"dupeOption": dupeOption,
	// 	"uuidOption": "preserve",
	// }

	result := &jobImportResults{}

	jobBytes, err := xml.Marshal(jobList)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := c.ProjectJobsImport(ctx, job.ProjectName, ioutil.NopCloser(bytes.NewReader(jobBytes)), "", "", "", dupeOption, "preserve")
	if err != nil {
		return nil, err
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal(respBytes, result)
	if err != nil {
		return nil, err
	}

	// err := c.postXMLBatch([]string{"jobs", "import"}, args, jobList, result)
	// if err != nil {
	// 	return nil, err
	// }

	if result.Failed.Count > 0 {
		errMsg := result.Failed.Results[0].Error
		return nil, fmt.Errorf(errMsg)
	}

	if result.Succeeded.Count != 1 {
		// Should never happen, since we send nothing in the request
		// that should cause a job to be skipped.
		return nil, fmt.Errorf("job was skipped")
	}

	return result.Succeeded.Results[0].JobSummary(), nil
}

// DeleteJob deletes the job with the given id.
// func (c *Client) DeleteJob(id string) error {
// 	return c.delete([]string{"job", id})
// }

func (c NotificationEmails) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if len(c) > 0 {
		return xml.Attr{Name: name, Value: strings.Join(c, ",")}, nil
	}
	return xml.Attr{}, nil
}

func (c *NotificationEmails) UnmarshalXMLAttr(attr xml.Attr) error {
	values := strings.Split(attr.Value, ",")
	*c = values
	return nil
}

func (c NotificationUrls) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if len(c) > 0 {
		return xml.Attr{Name: name, Value: strings.Join(c, ",")}, nil
	}
	return xml.Attr{}, nil
}

func (c *NotificationUrls) UnmarshalXMLAttr(attr xml.Attr) error {
	values := strings.Split(attr.Value, ",")
	*c = values
	return nil
}

func (c JobValueChoices) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if len(c) > 0 {
		return xml.Attr{Name: name, Value: strings.Join(c, ",")}, nil
	}
	return xml.Attr{}, nil
}

func (c *JobValueChoices) UnmarshalXMLAttr(attr xml.Attr) error {
	values := strings.Split(attr.Value, ",")
	*c = values
	return nil
}

func (a JobCommandJobRefArguments) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "line"}, Value: string(a)},
	}
	err := e.EncodeToken(start)
	if err != nil {
		return err
	}
	err = e.EncodeToken(xml.EndElement{Name: start.Name})
	return err
}

func (a *JobCommandJobRefArguments) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type jobRefArgs struct {
		Line string `xml:"line,attr"`
	}
	args := jobRefArgs{}
	err := d.DecodeElement(&args, &start)
	if err != nil {
		return err
	}

	*a = JobCommandJobRefArguments(args.Line)

	return nil
}

func (c JobPluginConfig) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	rc := map[string]string(c)
	return marshalMapToXML(&rc, e, start, "entry", "key", "value")
}

func (c *JobPluginConfig) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	rc := (*map[string]string)(c)
	return unmarshalMapFromXML(rc, d, start, "entry", "key", "value")
}

// Global log filter plugin configurations are marshalled differently than other plugins
func (config JobLogFilterConfig) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(config) == 0 {
		return nil
	}

	if err := e.EncodeToken(start); err != nil {
		return err
	}

	for key, value := range config {
		configEntry := xmlJobLogFilterConfigEntry{XMLName: xml.Name{Local: key}, Value: value}
		if err := e.Encode(configEntry); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
}

// Global log filter plugin configurations are unmarshalled differently than other plugins
func (config *JobLogFilterConfig) UnmarshalXML(d *xml.Decoder, _ xml.StartElement) error {
	*config = JobLogFilterConfig{}
	for {
		var entry xmlJobLogFilterConfigEntry

		if err := d.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		(*config)[entry.XMLName.Local] = entry.Value
	}
	return nil
}

// JobSummary produces a JobSummary instance with values populated from the import result.
// The summary object won't have its Description populated, since import results do not
// include descriptions.
func (r *jobImportResult) JobSummary() *JobSummary {
	return &JobSummary{
		ID:          r.ID,
		Name:        r.Name,
		GroupName:   r.GroupName,
		ProjectName: r.ProjectName,
	}
}
